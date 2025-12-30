package main

// HelpScreen shows the embedded help text in a read-only editor.
type HelpScreen struct {
	editor *Editor
}

func CreateHelpScreen(app *App, helpText string) (*HelpScreen, error) {
	editor := CreateEditor()
	editor.SetText(helpText)
	editor.SetReadOnly(true)

	hs := &HelpScreen{
		editor: editor,
	}
	return hs, nil
}

func (hs *HelpScreen) Render(app *App, ts *TileScreen) {
	screenPane := ts.GetPane()
	hs.editor.Render(screenPane, nil)
}

func (hs *HelpScreen) HandleKey(key Key) (KeyHandler, bool) {
	return hs.editor.HandleKey(key)
}

func (hs *HelpScreen) Reset() {
	hs.editor.Reset()
}

func (hs *HelpScreen) Close() {
}
