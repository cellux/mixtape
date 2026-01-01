package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MaxUndo = 64

// UndoFunc undoes an action.
type UndoFunc = func()

// UndoableFunction executes an action and tells how it can be undone.
type UndoableFunction = func() UndoFunc

type Action struct {
	doFunc     UndoableFunction // how to do it
	undoFunc   UndoFunc         // how to undo it
	pointAfter EditorPoint      // location of point right after the action
}

// EditScreen bundles the editor-related UI components.
type EditScreen struct {
	app         *App
	bm          *BufferManager
	editor      *Editor
	lastScript  []byte // last script successfully evaluated by VM
	lastBuffer  *Buffer
	tapeDisplay *TapeDisplay
	keymap      KeyMap

	fileBrowser     *FileBrowser // C-x f
	showFileBrowser bool

	bufferBrowser     *BufferBrowser // C-x b
	showBufferBrowser bool

	currentPrompt *Prompt
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
		bm:          app.bm,
		editor:      editor,
		tapeDisplay: tapeDisplay,
		keymap:      keymap,
	}
	es.editor.SetActionDispatcher(es.DispatchAction)

	es.syncBufferToEditor()

	fbFilter := func(fe FileEntry) bool {
		// show directories
		if fe.isDir {
			return true
		}
		// show files with .tape extension
		return filepath.Ext(fe.name) == ".tape"
	}

	fbStartDir := ""
	if path := es.GetCurrentBuffer().Path; path != "" {
		fbStartDir = filepath.Dir(path)
	}
	fb, err := CreateFileBrowser(fbStartDir, fbFilter, FileBrowserCallbacks{
		onSelect: es.handleFileBrowserSelection,
		onExit:   es.exitFileOpenMode,
	})
	if err != nil {
		return nil, err
	}
	es.fileBrowser = fb

	bb := CreateBufferBrowser(es.bm, BufferBrowserCallbacks{
		onSelect: es.handleBufferBrowserEnter,
		onExit:   es.exitBufferSwitchMode,
	})
	es.bufferBrowser = bb

	// eval editor script
	keymap.Bind("C-Enter", func() {
		es.syncEditorToBuffer()
		buf := es.GetCurrentBuffer()
		lastScript := buf.Data
		app.evalBuffer(buf, func() {
			es.lastScript = lastScript
		})
	})

	// eval if changed, then play
	keymap.Bind("C-p", func() {
		es.syncEditorToBuffer()
		buf := es.GetCurrentBuffer()
		if bytes.Equal(buf.Data, es.lastScript) {
			app.postEvent(func() {
				app.oto.PlayTape(app.vm.evalResult, es)
			}, false)
		} else {
			lastScript := buf.Data
			app.evalBuffer(buf, func() {
				es.lastScript = lastScript
				app.oto.PlayTape(app.vm.evalResult, es)
			})
		}
	})

	// save
	keymap.Bind("C-x s", func() {
		buf := es.GetCurrentBuffer()
		if !buf.HasPath() {
			es.openSavePrompt()
			return
		}
		es.SaveEditorContentToCurrentBuffer()
	})

	// save as
	keymap.Bind("C-x C-s", func() {
		es.openSavePrompt()
	})

	// file browser
	keymap.Bind("C-x f", func() {
		es.enterFileOpenMode()
	})

	// buffer browser
	keymap.Bind("C-x b", func() {
		es.enterBufferSwitchMode()
	})

	// switch to last buffer
	keymap.Bind("C-x o", func() {
		es.switchToOtherBuffer()
	})

	// switch to next buffer
	keymap.Bind("C-x n", func() {
		es.switchToAdjacentBuffer(1)
	})

	// switch to previous buffer
	keymap.Bind("C-x p", func() {
		es.switchToAdjacentBuffer(-1)
	})

	// kill current buffer
	keymap.Bind("C-x k", func() {
		if es.editor.Dirty() {
			// ask before we kill it
			es.openKillPrompt()
		} else {
			es.killCurrentBuffer()
		}
	})

	// undo
	keymap.Bind("C-z", func() { es.UndoLastAction() })
	keymap.Bind("C-x u", func() { es.UndoLastAction() })
	keymap.Bind("C-S--", func() { es.UndoLastAction() })

	return es, nil
}

func (es *EditScreen) GetCurrentBuffer() *Buffer {
	return es.bm.GetCurrentBuffer()
}

func (es *EditScreen) SetCurrentBuffer(b *Buffer) {
	es.lastBuffer = es.bm.GetCurrentBuffer()
	es.bm.SetCurrentBuffer(b)
}

func (es *EditScreen) DispatchAction(f UndoableFunction) {
	action := Action{doFunc: f}
	action.undoFunc = f()
	action.pointAfter = es.editor.GetPoint()
	buf := es.GetCurrentBuffer()
	buf.PushActionToUndoStack(action)
}

func (es *EditScreen) UndoLastAction() {
	buf := es.GetCurrentBuffer()
	if buf.UndoStackIsEmpty() {
		return
	}
	lastAction := buf.PopActionFromUndoStack()
	es.editor.SetPoint(lastAction.pointAfter)
	lastAction.undoFunc()
	es.editor.ForgetMark()
}

func (es *EditScreen) HandleKey(key Key) (next KeyHandler, handled bool) {
	// prompts behave like modal dialogs
	if es.currentPrompt != nil {
		next, handled = es.currentPrompt.HandleKey(key)
		return
	}
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

	currentBuffer := es.GetCurrentBuffer()

	var statusFile string
	if currentBuffer == nil {
		statusFile = "<no buffer>"
	} else if currentBuffer.HasPath() {
		statusFile = currentBuffer.Path
	} else {
		statusFile = currentBuffer.Name
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
	dirty := es.editor.Dirty() && currentBuffer.HasPath()
	es.editor.RenderStatusLine(
		editorStatusPane,
		statusFile,
		dirty,
		currentToken,
		app.rTotalFrames,
		app.rDoneFrames)

	if es.currentPrompt != nil {
		promptPane := screenPane.SubPane(0, screenPane.Height()-1, screenPane.Width(), 1)
		es.renderPrompt(promptPane)
		return
	}
}

func (es *EditScreen) switchToAdjacentBuffer(delta int) {
	adjacentBuffer := es.bm.getAdjacentBuffer(delta)
	if adjacentBuffer != nil {
		es.switchToBuffer(adjacentBuffer)
	}
}

func (es *EditScreen) switchToOtherBuffer() {
	if es.lastBuffer != nil {
		es.switchToBuffer(es.lastBuffer)
	} else {
		es.switchToAdjacentBuffer(1)
	}
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
	es.switchToBuffer(buf)
	es.exitBufferSwitchMode()
}

func (es *EditScreen) switchToBuffer(buf *Buffer) {
	if buf == nil {
		return
	}
	es.syncEditorToBuffer()
	es.SetCurrentBuffer(buf)
	es.syncBufferToEditor()
}

func (es *EditScreen) syncBufferToEditor() {
	currentBuffer := es.GetCurrentBuffer()
	es.editor.SetText(string(currentBuffer.Data))
	es.editor.point = currentBuffer.editorPoint
	es.editor.top = currentBuffer.editorTop
	es.editor.left = currentBuffer.editorLeft
	es.editor.dirty = currentBuffer.Dirty
	es.editor.Reset()
}

func (es *EditScreen) syncEditorToBuffer() {
	currentBuffer := es.GetCurrentBuffer()
	currentBuffer.SetData(es.editor.GetBytes())
	currentBuffer.editorPoint = es.editor.point
	currentBuffer.editorTop = es.editor.top
	currentBuffer.editorLeft = es.editor.left
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

func canonicalPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return canonical
}

func (es *EditScreen) loadFileToBuffer(path string, buf *Buffer) {
	data, err := os.ReadFile(path)
	if err != nil {
		es.app.SetLastError(err)
		return
	}
	if buf == nil {
		buf = es.bm.CreateBuffer("", path, data)
	} else {
		buf.SetData(data)
	}
	es.syncBufferToEditor()
	es.exitFileOpenMode()
}

func (es *EditScreen) handleFileBrowserSelection(entry FileEntry) {
	full := canonicalPath(entry.path)
	buf := es.bm.findBufferByPath(full)
	if buf == nil {
		es.loadFileToBuffer(full, nil)
		return
	}
	if !buf.Dirty {
		es.loadFileToBuffer(full, buf)
		return
	}
	prompt := CreateCharPrompt("Replace dirty buffer? (y/n)", "ynYN", PromptCallbacks{
		onConfirm: func(value string) {
			es.closePrompt()
			if value == "y" || value == "Y" {
				es.loadFileToBuffer(full, buf)
			}
		},
		onCancel: es.closePrompt,
	})
	es.openPrompt(prompt)
}

func (es *EditScreen) Reset() {
	es.editor.Reset()
	es.showBufferBrowser = false
	es.showFileBrowser = false
	es.currentPrompt = nil
}

func (es *EditScreen) openPrompt(prompt *Prompt) {
	es.currentPrompt = prompt
}

func (es *EditScreen) renderPrompt(tp TilePane) {
	if es.currentPrompt == nil {
		return
	}
	es.currentPrompt.Render(tp)
}

func (es *EditScreen) closePrompt() {
	es.currentPrompt = nil
}

func (es *EditScreen) openSavePrompt() {
	cwd, err := os.Getwd()
	if err != nil {
		es.app.SetLastError(err)
		return
	}
	defaultPath := cwd
	currentBuffer := es.GetCurrentBuffer()
	if currentBuffer.HasPath() {
		defaultPath = currentBuffer.Path
	} else if !strings.HasSuffix(defaultPath, string(filepath.Separator)) {
		defaultPath += string(filepath.Separator)
	}
	prompt := CreateTextPrompt("Save file: ", PromptCallbacks{
		onConfirm: es.confirmSavePrompt,
		onCancel:  es.closePrompt,
	})
	prompt.SetText(defaultPath)
	es.openPrompt(prompt)
}

func (es *EditScreen) SaveEditorContentToCurrentBuffer() {
	es.syncEditorToBuffer()
	currentBuffer := es.GetCurrentBuffer()
	if err := currentBuffer.WriteFile(); err != nil {
		es.app.SetLastError(err)
	}
	es.syncBufferToEditor()
}

func (es *EditScreen) confirmSavePrompt(value string) {
	es.closePrompt()
	path := value
	if path == "" {
		return
	}
	currentBuffer := es.GetCurrentBuffer()
	currentBuffer.SetPath(path)
	es.SaveEditorContentToCurrentBuffer()
}

func (es *EditScreen) openKillPrompt() {
	prompt := CreateCharPrompt("Kill buffer? (y/n)", "ynYN", PromptCallbacks{
		onConfirm: es.confirmKillPrompt,
		onCancel:  es.closePrompt,
	})
	es.openPrompt(prompt)
}

func (es *EditScreen) killCurrentBuffer() {
	if len(es.bm.buffers) == 1 {
		es.app.Quit()
		return
	}
	nextBuffer := es.lastBuffer
	if nextBuffer == nil {
		nextBuffer = es.bm.getAdjacentBuffer(1)
	}
	target := es.GetCurrentBuffer()
	es.bm.RemoveBuffer(target)
	if es.lastBuffer == target {
		es.lastBuffer = nil
	}
	if nextBuffer == target || nextBuffer == nil {
		nextBuffer = es.bm.FirstBuffer()
	}
	es.SetCurrentBuffer(nextBuffer)
	es.syncBufferToEditor()
}

func (es *EditScreen) confirmKillPrompt(value string) {
	es.closePrompt()
	if value == "y" || value == "Y" {
		es.killCurrentBuffer()
	}
}

func (es *EditScreen) OnChar(app *App, char rune) {
	if es.currentPrompt != nil {
		es.currentPrompt.OnChar(char)
		return
	}
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
