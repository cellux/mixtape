package main

import (
	"fmt"
	"os"
	"slices"
)

// EditScreen bundles the editor-related UI components.
type EditScreen struct {
	app         *App
	editor      *Editor
	lastScript  []byte // last script successfully evaluated by VM
	tapeDisplay *TapeDisplay
	keymap      KeyMap
}

func CreateEditScreen(app *App) (*EditScreen, error) {
	editor := CreateEditor()
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}

	keymap := CreateKeyMap()

	es := &EditScreen{
		app:         app,
		editor:      editor,
		tapeDisplay: tapeDisplay,
		keymap:      keymap,
	}
	es.loadCurrentBufferIntoEditor()

	keymap.Bind("Enter", func() {
		DispatchAction(func() UndoFunc {
			editor.SplitLine()
			return func() {
				editor.AdvanceColumn(-1)
				editor.DeleteRune()
			}
		})
	})
	keymap.Bind("Left", func() {
		editor.AdvanceColumn(-1)
	})
	keymap.Bind("Right", func() {
		editor.AdvanceColumn(1)
	})
	keymap.Bind("Up", func() {
		editor.AdvanceLine(-1)
	})
	keymap.Bind("Down", func() {
		editor.AdvanceLine(1)
	})
	keymap.Bind("PageUp", func() {
		for range editor.height {
			editor.AdvanceLine(-1)
		}
	})
	keymap.Bind("PageDown", func() {
		for range editor.height {
			editor.AdvanceLine(1)
		}
	})
	keymap.Bind("Delete", func() {
		DispatchAction(func() UndoFunc {
			deletedRune := editor.DeleteRune()
			return func() {
				if deletedRune != 0 {
					editor.InsertRune(deletedRune)
					editor.AdvanceColumn(-1)
				}
			}
		})
	})
	keymap.Bind("Backspace", func() {
		if editor.AtBOF() {
			return
		}
		DispatchAction(func() UndoFunc {
			editor.AdvanceColumn(-1)
			deletedRune := editor.DeleteRune()
			return func() {
				if deletedRune != 0 {
					editor.InsertRune(deletedRune)
				}
			}
		})
	})
	keymap.Bind("Home", editor.MoveToBOL)
	keymap.Bind("End", editor.MoveToEOL)
	keymap.Bind("Tab", editor.InsertSpacesUntilNextTabStop)
	keymap.Bind("C-Enter", func() {
		editorScript := editor.GetBytes()
		app.evalEditorScript(editorScript, func() {
			es.lastScript = editorScript
		})
	})
	keymap.Bind("C-p", func() {
		editorScript := editor.GetBytes()
		if slices.Compare(editorScript, es.lastScript) != 0 {
			app.evalEditorScript(editorScript, func() {
				es.lastScript = editorScript
				app.oto.PlayTape(app.vm.evalResult, es)
			})
		} else {
			app.postEvent(func() {
				app.oto.PlayTape(app.vm.evalResult, es)
			}, false)
		}
	})
	keymap.Bind("C-Left", editor.WordLeft)
	keymap.Bind("C-Right", editor.WordRight)
	keymap.Bind("C-a", editor.MoveToBOL)
	keymap.Bind("C-e", editor.MoveToEOL)
	keymap.Bind("C-Home", editor.MoveToBOF)
	keymap.Bind("C-End", editor.MoveToEOF)
	keymap.Bind("C-k", func() {
		if editor.AtEOL() {
			editor.DeleteRune()
		} else {
			for !editor.AtEOL() {
				editor.DeleteRune()
			}
		}
	})
	keymap.Bind("C-Backspace", func() {
		DispatchAction(func() UndoFunc {
			editor.SetMark()
			editor.WordLeft()
			deletedRunes := editor.KillRegion()
			return func() {
				editor.InsertRunes(deletedRunes)
			}
		})
	})
	keymap.Bind("C-u", func() {
		DispatchAction(func() UndoFunc {
			editor.SetMark()
			editor.MoveToBOL()
			deletedRunes := editor.KillRegion()
			return func() {
				editor.InsertRunes(deletedRunes)
			}
		})
	})
	keymap.Bind("C-Space", editor.SetMark)
	keymap.Bind("C-w", func() {
		DispatchAction(func() UndoFunc {
			p, _ := editor.PointAndMarkInOrder()
			deletedRunes := editor.KillRegion()
			return func() {
				editor.SetPoint(p)
				editor.InsertRunes(deletedRunes)
			}
		})
	})
	keymap.Bind("C-y", func() {
		DispatchAction(func() UndoFunc {
			p0 := editor.GetPoint()
			editor.Paste()
			p1 := editor.GetPoint()
			return func() {
				editor.KillBetween(p0, p1)
			}
		})
	})
	keymap.Bind("C-x C-s", func() {
		es.syncEditorToBuffer()
		if app.currentBuffer != nil && app.currentBuffer.HasPath() {
			if err := os.WriteFile(app.currentBuffer.Path, editor.GetBytes(), 0o644); err != nil {
				app.SetLastError(err)
			}
		}
	})
	keymap.Bind("M-b", editor.WordLeft)
	keymap.Bind("M-f", editor.WordRight)
	keymap.Bind("M-w", editor.YankRegion)
	keymap.Bind("M-Backspace", func() {
		editor.SetMark()
		editor.WordLeft()
		editor.KillRegion()
	})

	return es, nil
}

func (es *EditScreen) HandleKey(key Key) (KeyHandler, bool) {
	return es.keymap.HandleKey(key)
}

func (es *EditScreen) Render(app *App, ts *TileScreen) {
	screenPane := ts.GetPane()

	var statusFile string
	if app.currentBuffer == nil {
		statusFile = "<no buffer>"
	} else if !app.currentBuffer.HasPath() {
		statusFile = app.currentBuffer.Name
	} else {
		statusFile = app.currentBuffer.Path
	}

	var editorPane TilePane
	var tapeDisplayPane TilePane
	var statusPane TilePane
	var errorPane TilePane

	if err := app.vm.errResult; err != nil {
		screenPane, errorPane = screenPane.SplitY(-1)
		errorPane.WithFgBg(ColorWhite, ColorRed, func() {
			errorPane.DrawString(0, 0, err.Error())
		})
	}

	switch result := app.vm.evalResult.(type) {
	case *Tape:
		editorPane, tapeDisplayPane = screenPane.SplitY(-8)
		var playheadFrames []int
		for _, tp := range app.oto.GetTapePlayers(es) {
			playheadFrames = append(playheadFrames, tp.GetCurrentFrame())
		}
		es.tapeDisplay.Render(result, tapeDisplayPane.GetPixelRect(), result.nframes, 0, playheadFrames)
	default:
		if result == nil {
			editorPane = screenPane
		} else {
			editorPane, statusPane = screenPane.SplitY(-1)
			statusPane.DrawString(0, 0, fmt.Sprintf("%#v", result))
		}
	}

	editorBufferPane, editorStatusPane := editorPane.SplitY(-1)
	currentToken := app.vm.CurrentToken()
	es.editor.Render(editorBufferPane, currentToken)
	if app.lastError != nil {
		editorStatusPane.WithFgBg(ColorWhite, ColorRed, func() {
			editorStatusPane.DrawString(0, 0, app.lastError.Error())
		})
	} else {
		es.editor.RenderStatusLine(editorStatusPane, statusFile, currentToken, app.rTotalFrames, app.rDoneFrames)
	}
}

func (es *EditScreen) syncEditorToBuffer() {
	if es.app == nil || es.app.currentBuffer == nil {
		return
	}
	es.app.currentBuffer.Data = es.editor.GetBytes()
}

func (es *EditScreen) Keymap() KeyMap {
	return es.keymap
}

func (es *EditScreen) Reset() {
	es.editor.Reset()
}

func (es *EditScreen) loadCurrentBufferIntoEditor() {
	if es.app != nil && es.app.currentBuffer != nil {
		es.editor.SetText(string(es.app.currentBuffer.Data))
	}
}

func (es *EditScreen) OnChar(app *App, char rune) {
	DispatchAction(func() UndoFunc {
		es.editor.InsertRune(char)
		return func() {
			es.editor.AdvanceColumn(-1)
			es.editor.DeleteRune()
		}
	})
}

func (es *EditScreen) Close() {
}
