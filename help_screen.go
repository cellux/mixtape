package main

// HelpScreen shows the embedded help text in a read-only editor.
type HelpScreen struct {
	editor *Editor
	keymap KeyMap
}

func CreateHelpScreen(app *App, parent KeyMap, helpText string) (*HelpScreen, error) {
	editor := CreateEditor()
	editor.SetText(helpText)
	editor.SetReadOnly(true)

	keymap := CreateKeyMap(parent)

	// Navigation
	keymap.Bind("Left", func() { editor.AdvanceColumn(-1) })
	keymap.Bind("Right", func() { editor.AdvanceColumn(1) })
	keymap.Bind("Up", func() { editor.AdvanceLine(-1) })
	keymap.Bind("Down", func() { editor.AdvanceLine(1) })
	keymap.Bind("Home", editor.MoveToBOL)
	keymap.Bind("End", editor.MoveToEOL)
	keymap.Bind("C-Home", editor.MoveToBOF)
	keymap.Bind("C-End", editor.MoveToEOF)
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
	keymap.Bind("C-Left", editor.WordLeft)
	keymap.Bind("C-Right", editor.WordRight)
	keymap.Bind("M-b", editor.WordLeft)
	keymap.Bind("M-f", editor.WordRight)
	keymap.Bind("C-a", editor.MoveToBOL)
	keymap.Bind("C-e", editor.MoveToEOL)
	keymap.Bind("C-Space", editor.SetMark)
	keymap.Bind("M-w", editor.YankRegion)

	hs := &HelpScreen{
		editor: editor,
		keymap: keymap,
	}
	return hs, nil
}

func (hs *HelpScreen) Render(app *App, ts *TileScreen) {
	screenPane := ts.GetPane()
	hs.editor.Render(screenPane, nil)
}

func (hs *HelpScreen) Keymap() KeyMap {
	return hs.keymap
}

func (hs *HelpScreen) Reset() {
	hs.editor.Reset()
}

func (hs *HelpScreen) Close() {
}
