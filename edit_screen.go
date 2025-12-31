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
	es.editor.SetActionDispatcher(es.DispatchAction)

	tapeFilter := func(fe FileEntry) bool {
		if fe.isDir {
			return true
		}
		return filepath.Ext(fe.name) == ".tape"
	}

	fb, err := CreateFileBrowser(app, "", tapeFilter, FileBrowserCallbacks{
		onSelect: es.handleFileBrowserSelection,
		onExit:   es.exitFileOpenMode,
	})
	if err != nil {
		return nil, err
	}
	es.fileBrowser = fb

	bb := CreateBufferBrowser(app, BufferBrowserCallbacks{
		onSelect: es.handleBufferBrowserEnter,
		onExit:   es.exitBufferSwitchMode,
	})
	es.bufferBrowser = bb

	es.loadCurrentBufferIntoEditor()

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

	keymap.Bind("C-x s", func() {
		es.syncEditorToBuffer()
		if app.currentBuffer != nil && app.currentBuffer.HasPath() {
			if err := os.WriteFile(app.currentBuffer.Path, editor.GetBytes(), 0o644); err != nil {
				app.SetLastError(err)
			}
		}
	})
	keymap.Bind("C-x f", func() {
		es.enterFileOpenMode()
	})
	keymap.Bind("C-x b", func() {
		es.enterBufferSwitchMode()
	})
	keymap.Bind("C-x o", func() {
		es.switchToBuffer(es.app.lastBuffer)
	})
	keymap.Bind("C-x n", func() {
		es.switchToAdjacentBuffer(1)
	})
	keymap.Bind("C-x p", func() {
		es.switchToAdjacentBuffer(-1)
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

func (es *EditScreen) HandleKey(key Key) (next KeyHandler, handled bool) {
	if es.showFileBrowser {
		next, handled = es.fileBrowser.HandleKey(key)
		if handled {
			return
		}
	}
	if es.showBufferBrowser {
		next, handled = es.bufferBrowser.HandleKey(key)
		if handled {
			return
		}
	}
	next, handled = es.editor.HandleKey(key)
	if handled {
		return
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

func (es *EditScreen) switchToAdjacentBuffer(delta int) {
	n := len(es.app.buffers)
	if n < 2 {
		return
	}
	currentIndex := -1
	for i, buf := range es.app.buffers {
		if buf == es.app.currentBuffer {
			currentIndex = i
			break
		}
	}
	if currentIndex == -1 {
		return
	}
	nextIndex := (currentIndex + delta + n) % n
	es.switchToBuffer(es.app.buffers[nextIndex])
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
