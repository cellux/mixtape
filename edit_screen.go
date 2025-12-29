package main

import (
	"fmt"
	"os"
	"slices"
)

// EditScreen bundles the editor-related UI components.
type EditScreen struct {
	editor      *Editor
	lastScript  []byte // last script successfully evaluated by VM
	tapeDisplay *TapeDisplay
	keymap      KeyMap
}

func CreateEditScreen(app *App, parent KeyMap, initialText string) (*EditScreen, error) {
	editor := CreateEditor()
	editor.SetText(initialText)
	tapeDisplay, err := CreateTapeDisplay()
	if err != nil {
		return nil, err
	}

	keymap := CreateKeyMap(parent)

	es := &EditScreen{
		editor:      editor,
		tapeDisplay: tapeDisplay,
		keymap:      keymap,
	}

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
				app.playEvalResult()
			})
		} else {
			app.postEvent(app.playEvalResult, false)
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
		if app.currentFile != "" {
			os.WriteFile(app.currentFile, editor.GetBytes(), 0o644)
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

func (es *EditScreen) Render(app *App, ts *TileScreen) {
	screenPane := ts.GetPane()

	var statusFile string
	if app.currentFile == "" {
		statusFile = "<no file>"
	} else {
		statusFile = app.currentFile
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
		for _, tp := range app.oto.GetTapePlayers() {
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
	es.editor.RenderStatusLine(editorStatusPane, statusFile, currentToken, app.rTotalFrames, app.rDoneFrames)
}

func (es *EditScreen) Keymap() KeyMap {
	return es.keymap
}

func (es *EditScreen) Reset() {
	es.editor.Reset()
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
