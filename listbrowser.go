package main

import "fmt"

// ListBrowser adds common searchable-list interaction and header rendering to
// a ListDisplay. Its callbacks leave data loading and selection behavior to
// the owning browser.
type ListBrowserCallbacks struct {
	onEnter                  func()
	onExit                   func()
	onBackspaceWithoutFilter func()
}

type ListBrowser struct {
	*ListDisplay
	header    func() string
	keymap    KeyMap
	callbacks ListBrowserCallbacks
}

func CreateListBrowser(header func() string, callbacks ListBrowserCallbacks) *ListBrowser {
	lb := &ListBrowser{
		ListDisplay: CreateListDisplay(),
		header:      header,
		callbacks:   callbacks,
	}
	lb.initKeymap()
	return lb
}

func (lb *ListBrowser) initKeymap() {
	lb.keymap = CreateKeyMap()
	lb.keymap.Bind("Up", func() { lb.MoveBy(-1) })
	lb.keymap.Bind("Down", func() { lb.MoveBy(1) })
	lb.keymap.Bind("Home", func() { lb.MoveTo(0) })
	lb.keymap.Bind("End", lb.MoveToEnd)
	lb.keymap.Bind("PageUp", func() { lb.MoveBy(-lb.PageSize()) })
	lb.keymap.Bind("PageDown", func() { lb.MoveBy(lb.PageSize()) })
	lb.keymap.Bind("Backspace", lb.handleBackspace)
	lb.keymap.Bind("Enter", lb.handleEnter)
	lb.keymap.Bind("Escape", lb.handleEscape)
	lb.keymap.Bind("C-g", lb.exit)
}

func (lb *ListBrowser) Keymap() KeyMap {
	return lb.keymap
}

func (lb *ListBrowser) HandleKey(key Key) (KeyHandler, bool) {
	return lb.keymap.HandleKey(key)
}

func (lb *ListBrowser) MoveToEnd() {
	lb.MoveTo(len(lb.GetFilteredEntries()) - 1)
}

func (lb *ListBrowser) OnChar(char rune) {
	lb.AppendSearchChar(char)
}

func (lb *ListBrowser) handleBackspace() {
	if lb.FilterMode() {
		lb.RemoveLastSearchChar()
		return
	}
	if lb.callbacks.onBackspaceWithoutFilter != nil {
		lb.callbacks.onBackspaceWithoutFilter()
	}
}

func (lb *ListBrowser) handleEnter() {
	if lb.callbacks.onEnter != nil {
		lb.callbacks.onEnter()
	}
}

func (lb *ListBrowser) handleEscape() {
	if lb.FilterMode() {
		lb.Reset()
		return
	}
	lb.exit()
}

func (lb *ListBrowser) exit() {
	if lb.callbacks.onExit != nil {
		lb.callbacks.onExit()
	}
}

func (lb *ListBrowser) Render(tp TilePane) {
	if tp.Height() <= 0 {
		return
	}

	headerText := ""
	if lb.header != nil {
		headerText = lb.header()
	}
	header := tp.SubPane(0, 0, tp.Width(), 1)
	header.DrawString(0, 0, headerText)
	if lb.FilterMode() {
		header.WithFgBg(ColorWhite, ColorGreen, func() {
			header.DrawString(len(headerText)+1, 0, fmt.Sprintf("[%s]", lb.SearchText()))
		})
	}

	listPane := tp.SubPane(0, 1, tp.Width(), tp.Height()-1)
	lb.ListDisplay.Render(listPane)
}
