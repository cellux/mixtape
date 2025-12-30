package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const MaxUndo = 64

type UndoFunc = func()
type UndoableFunction = func() UndoFunc

type Action struct {
	doFunc     UndoableFunction
	undoFunc   UndoFunc
	pointAfter EditorPoint
}

// EditScreen bundles the editor-related UI components.
type EditScreen struct {
	app         *App
	editor      *Editor
	lastScript  []byte // last script successfully evaluated by VM
	tapeDisplay *TapeDisplay
	keymap      KeyMap

	fileBrowser     *FileBrowser
	showFileBrowser bool

	bufferBrowser     *BufferBrowser
	showBufferBrowser bool
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
	}, es.handleFileBrowserSelection, es.exitFileOpenMode)
	if err != nil {
		return nil, err
	}
	es.fileBrowser = fb

	bb := CreateBufferBrowser(app, es.handleBufferBrowserEnter, es.exitBufferSwitchMode)
	es.bufferBrowser = bb

	es.loadCurrentBufferIntoEditor()

	keymap.Bind("Enter", func() {
		es.DispatchAction(func() UndoFunc {
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
		es.DispatchAction(func() UndoFunc {
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
		es.DispatchAction(func() UndoFunc {
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
	keymap.Bind("Tab", func() {
		es.DispatchAction(func() UndoFunc {
			start := editor.GetPoint()
			editor.InsertSpacesUntilNextTabStop()
			end := editor.GetPoint()
			inserted := end.column - start.column
			return func() {
				if inserted <= 0 {
					return
				}
				editor.SetPoint(end)
				for range inserted {
					editor.AdvanceColumn(-1)
					editor.DeleteRune()
				}
				editor.SetPoint(start)
			}
		})
	})
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
		es.DispatchAction(func() UndoFunc {
			start := editor.GetPoint()
			var deletedRunes []rune
			if editor.AtEOL() {
				if r := editor.DeleteRune(); r != 0 {
					deletedRunes = append(deletedRunes, r)
				}
			} else {
				for !editor.AtEOL() {
					if r := editor.DeleteRune(); r != 0 {
						deletedRunes = append(deletedRunes, r)
					}
				}
			}
			return func() {
				if len(deletedRunes) == 0 {
					return
				}
				editor.SetPoint(start)
				editor.InsertRunes(deletedRunes)
				editor.SetPoint(start)
			}
		})
	})
	keymap.Bind("C-Backspace", func() {
		es.DispatchAction(func() UndoFunc {
			editor.SetMark()
			editor.WordLeft()
			deletedRunes := editor.KillRegion()
			return func() {
				editor.InsertRunes(deletedRunes)
			}
		})
	})
	keymap.Bind("C-u", func() {
		es.DispatchAction(func() UndoFunc {
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
		es.DispatchAction(func() UndoFunc {
			start := editor.GetPoint()
			p, _ := editor.PointAndMarkInOrder()
			deletedRunes := editor.KillRegion()
			return func() {
				editor.SetPoint(p)
				editor.InsertRunes(deletedRunes)
				editor.SetPoint(start)
			}
		})
	})
	keymap.Bind("C-y", func() {
		es.DispatchAction(func() UndoFunc {
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
	keymap.Bind("C-x b", func() {
		es.switchToBuffer(es.app.lastBuffer)
	})
	keymap.Bind("C-x C-b", func() {
		es.enterBufferSwitchMode()
	})
	keymap.Bind("M-b", editor.WordLeft)
	keymap.Bind("M-f", editor.WordRight)
	keymap.Bind("M-w", editor.YankRegion)
	keymap.Bind("M-Backspace", func() {
		es.DispatchAction(func() UndoFunc {
			editor.SetMark()
			editor.WordLeft()
			p, _ := editor.PointAndMarkInOrder()
			deletedRunes := editor.KillRegion()
			return func() {
				if len(deletedRunes) == 0 {
					return
				}
				editor.SetPoint(p)
				editor.InsertRunes(deletedRunes)
			}
		})
	})
	keymap.Bind("C-z", func() { es.UndoLastAction() })
	keymap.Bind("C-x u", func() { es.UndoLastAction() })
	keymap.Bind("C-S--", func() { es.UndoLastAction() })

	return es, nil
}

func (es *EditScreen) DispatchAction(f UndoableFunction) {
	editor := es.editor
	buf := es.app.currentBuffer

	action := Action{doFunc: f}
	action.undoFunc = f()
	action.pointAfter = editor.GetPoint()

	buf.undoStack = append(buf.undoStack, action)
	if len(buf.undoStack) > MaxUndo {
		buf.undoStack = slices.Delete(buf.undoStack, 0, len(buf.undoStack)-MaxUndo)
	}
}

func (es *EditScreen) UndoLastAction() {
	editor := es.editor
	buf := es.app.currentBuffer

	if len(buf.undoStack) == 0 {
		return
	}

	lastAction := buf.undoStack[len(buf.undoStack)-1]
	buf.undoStack = buf.undoStack[:len(buf.undoStack)-1]
	editor.SetPoint(lastAction.pointAfter)
	editor.ForgetMark()
	lastAction.undoFunc()
}

func (es *EditScreen) HandleKey(key Key) (KeyHandler, bool) {
	if es.showFileBrowser {
		next, handled := es.fileBrowser.HandleKey(key)
		if handled {
			return next, true
		}
	}
	if es.showBufferBrowser {
		next, handled := es.bufferBrowser.HandleKey(key)
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

	if es.showFileBrowser {
		es.fileBrowser.Render(editorPane)
		return
	}
	if es.showBufferBrowser {
		es.bufferBrowser.Render(editorPane)
		return
	}

	editorBufferPane, editorStatusPane := editorPane.SplitY(-1)
	currentToken := app.vm.CurrentToken()
	es.editor.Render(editorBufferPane, currentToken)
	es.editor.RenderStatusLine(editorStatusPane, statusFile, currentToken, app.rTotalFrames, app.rDoneFrames)
}

func (es *EditScreen) enterBufferSwitchMode() {
	es.syncEditorToBuffer()
	es.bufferBrowser.Reset()
	es.showBufferBrowser = true
}

func (es *EditScreen) exitBufferSwitchMode() {
	es.showBufferBrowser = false
}

func (es *EditScreen) handleBufferBrowserEnter(buf *Buffer) {
	if buf == nil {
		return
	}
	es.switchToBuffer(buf)
	es.exitBufferSwitchMode()
}

func (es *EditScreen) switchToBuffer(buf *Buffer) {
	if buf == nil {
		return
	}
	es.syncEditorToBuffer()
	if es.app.currentBuffer != nil && es.app.currentBuffer != buf {
		es.app.lastBuffer = es.app.currentBuffer
	}
	es.app.currentBuffer = buf
	es.loadCurrentBufferIntoEditor()
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

func (es *EditScreen) handleFileBrowserSelection(entry FileEntry) {
	if entry.isDir {
		if _, err := es.fileBrowser.Enter(); err != nil {
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
	buf := CreateBuffer(es.app.buffers, full, data)
	es.app.buffers = append(es.app.buffers, buf)
	es.app.currentBuffer = buf
	es.loadCurrentBufferIntoEditor()
	es.exitFileOpenMode()
}

func (es *EditScreen) Reset() {
	es.editor.Reset()
	es.showBufferBrowser = false
	es.showFileBrowser = false
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
	if es.showBufferBrowser {
		es.bufferBrowser.OnChar(char)
		return
	}
	es.DispatchAction(func() UndoFunc {
		es.editor.InsertRune(char)
		return func() {
			es.editor.AdvanceColumn(-1)
			es.editor.DeleteRune()
		}
	})
}

func (es *EditScreen) Close() {
}
