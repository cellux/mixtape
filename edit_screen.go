package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// EditScreen bundles the editor-related UI components.
type EditScreen struct {
	app         *App
	editor      *Editor
	lastScript  []byte // last script successfully evaluated by VM
	tapeDisplay *TapeDisplay
	keymap      KeyMap

	fileBrowser       *FileBrowser
	fileBrowserKeymap KeyMap
	showFileBrowser   bool
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

	fb, err := CreateFileBrowser(app, "", func(fe FileEntry) bool {
		if fe.isDir {
			return true
		}
		return filepath.Ext(fe.name) == ".tape"
	})
	if err != nil {
		return nil, err
	}
	es.fileBrowser = fb

	fbKeyMap := CreateKeyMap()
	fbKeyMap.Bind("Up", func() { es.fileBrowser.MoveBy(-1) })
	fbKeyMap.Bind("Down", func() { es.fileBrowser.MoveBy(1) })
	fbKeyMap.Bind("Home", func() { es.fileBrowser.MoveTo(0) })
	fbKeyMap.Bind("End", func() { es.fileBrowser.MoveToEnd() })
	fbKeyMap.Bind("PageUp", func() { es.fileBrowser.MoveBy(-es.fileBrowser.PageSize()) })
	fbKeyMap.Bind("PageDown", func() { es.fileBrowser.MoveBy(es.fileBrowser.PageSize()) })
	fbKeyMap.Bind("Enter", func() { es.handleFileBrowserEnter() })
	fbKeyMap.Bind("Escape", es.exitFileOpenMode)
	fbKeyMap.Bind("C-g", es.exitFileOpenMode)
	fbKeyMap.Bind("Backspace", func() { es.handleFileBrowserBackspace() })
	es.fileBrowserKeymap = fbKeyMap

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
	keymap.Bind("C-x C-f", func() {
		es.enterFileOpenMode()
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
	if es.showFileBrowser {
		next, handled := es.fileBrowserKeymap.HandleKey(key)
		if handled {
			return next, true
		}
	}
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

	var browserPane TilePane
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

	if es.showFileBrowser {
		browserPane, screenPane = screenPane.SplitY(8)
		browserPane.DrawString(0, 0, es.fileBrowser.Directory())
		if es.fileBrowser.SearchText() != "" {
			browserPane.WithFgBg(ColorWhite, ColorGreen, func() {
				browserPane.DrawString(len(es.fileBrowser.Directory())+1, 0, fmt.Sprintf("[%s]", es.fileBrowser.SearchText()))
			})
		}
		listPane := browserPane.SubPane(0, 1, browserPane.Width(), browserPane.Height()-1)
		es.fileBrowser.listDisplay.lastHeight = listPane.Height()
		es.fileBrowser.Render(&listPane)
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
	es.editor.RenderStatusLine(editorStatusPane, statusFile, currentToken, app.rTotalFrames, app.rDoneFrames)
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

func (es *EditScreen) enterFileOpenMode() {
	if err := es.fileBrowser.Reset(); err != nil {
		es.app.SetLastError(err)
	}
	es.showFileBrowser = true
}

func (es *EditScreen) exitFileOpenMode() {
	es.showFileBrowser = false
}

func (es *EditScreen) handleFileBrowserBackspace() {
	_, err := es.fileBrowser.HandleBackspace()
	if err != nil {
		es.app.SetLastError(err)
	}
}

func (es *EditScreen) handleFileBrowserEnter() {
	entry := es.fileBrowser.CurrentFilteredEntry()
	if entry == nil {
		return
	}
	if entry.isDir {
		_, err := es.fileBrowser.Enter()
		if err != nil {
			es.app.SetLastError(err)
		}
		return
	}
	full := es.fileBrowser.CanonicalPath(entry.path)
	if buf := es.app.findBufferByPath(full); buf != nil {
		es.app.SetLastError(fmt.Errorf("buffer already exists for %s", full))
		es.exitFileOpenMode()
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		es.app.SetLastError(err)
		return
	}
	buf := &Buffer{Name: filepath.Base(full), Path: full, Data: data}
	es.app.buffers = append(es.app.buffers, buf)
	es.app.currentBuffer = buf
	es.loadCurrentBufferIntoEditor()
	es.exitFileOpenMode()
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
	if es.showFileBrowser {
		es.fileBrowser.OnChar(char)
		return
	}
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
