package main

import (
	"slices"
	"unicode"
)

type InputFieldCallbacks struct {
	onConfirm func()
	onCancel  func()
}

type InputField struct {
	runes     []rune
	point     int
	left      int
	keymap    KeyMap
	callbacks InputFieldCallbacks
}

func CreateInputField(callbacks InputFieldCallbacks) *InputField {
	f := &InputField{callbacks: callbacks}
	f.initKeymap()
	return f
}

func (f *InputField) SetCallbacks(callbacks InputFieldCallbacks) {
	f.callbacks = callbacks
}

func (f *InputField) initKeymap() {
	f.keymap = CreateKeyMap()

	f.keymap.Bind("Left", func() { f.AdvanceColumn(-1) })
	f.keymap.Bind("Right", func() { f.AdvanceColumn(1) })
	f.keymap.Bind("Home", f.MoveToBOL)
	f.keymap.Bind("End", f.MoveToEOL)

	f.keymap.Bind("C-Left", f.WordLeft)
	f.keymap.Bind("C-Right", f.WordRight)
	f.keymap.Bind("M-b", f.WordLeft)
	f.keymap.Bind("M-f", f.WordRight)
	f.keymap.Bind("C-a", f.MoveToBOL)
	f.keymap.Bind("C-e", f.MoveToEOL)

	f.keymap.Bind("Backspace", func() { f.Backspace() })
	f.keymap.Bind("Delete", func() { f.DeleteRune() })
	f.keymap.Bind("C-k", func() { f.KillToEnd() })
	f.keymap.Bind("Enter", func() {
		if f.callbacks.onConfirm != nil {
			f.callbacks.onConfirm()
		}
	})
	f.keymap.Bind("Escape", func() {
		if f.callbacks.onCancel != nil {
			f.callbacks.onCancel()
		}
	})
	f.keymap.Bind("C-g", func() {
		if f.callbacks.onCancel != nil {
			f.callbacks.onCancel()
		}
	})
}

func (f *InputField) HandleKey(key Key) (KeyHandler, bool) {
	return f.keymap.HandleKey(key)
}

func (f *InputField) Keymap() KeyMap {
	return f.keymap
}

func (f *InputField) SetText(text string) {
	f.runes = []rune(text)
	f.point = len(f.runes)
	f.left = 0
}

func (f *InputField) Text() string {
	return string(f.runes)
}

func (f *InputField) Length() int {
	return len(f.runes)
}

func (f *InputField) AtBOL() bool {
	return f.point == 0
}

func (f *InputField) AtEOL() bool {
	return f.point == len(f.runes)
}

func (f *InputField) CurrentRune() rune {
	if f.AtEOL() {
		return 0
	}
	return f.runes[f.point]
}

func (f *InputField) AdvanceColumn(amount int) {
	f.point += amount
	if f.point < 0 {
		f.point = 0
	}
	if f.point > len(f.runes) {
		f.point = len(f.runes)
	}
}

func (f *InputField) MoveToBOL() {
	f.point = 0
}

func (f *InputField) MoveToEOL() {
	f.point = len(f.runes)
}

func (f *InputField) WordLeft() {
	if !f.AtBOL() {
		f.AdvanceColumn(-1)
	}
	for !f.AtBOL() && !isWordConstituent(f.CurrentRune()) {
		f.AdvanceColumn(-1)
	}
	m := f.point
	for !f.AtBOL() && isWordConstituent(f.CurrentRune()) {
		f.AdvanceColumn(-1)
	}
	if f.point != m && !isWordConstituent(f.CurrentRune()) {
		f.AdvanceColumn(1)
	}
}

func (f *InputField) WordRight() {
	for !f.AtEOL() && unicode.IsSpace(f.CurrentRune()) {
		f.AdvanceColumn(1)
	}
	for !f.AtEOL() && !isWordConstituent(f.CurrentRune()) {
		f.AdvanceColumn(1)
	}
	for !f.AtEOL() && isWordConstituent(f.CurrentRune()) {
		f.AdvanceColumn(1)
	}
}

func (f *InputField) InsertRune(r rune) {
	if r == '\n' || r == '\r' {
		return
	}
	f.runes = slices.Insert(f.runes, f.point, r)
	f.AdvanceColumn(1)
}

func (f *InputField) InsertRunes(rs []rune) {
	for _, r := range rs {
		f.InsertRune(r)
	}
}

func (f *InputField) DeleteRune() (deleted rune) {
	if f.AtEOL() {
		return 0
	}
	deleted = f.runes[f.point]
	f.runes = slices.Delete(f.runes, f.point, f.point+1)
	return deleted
}

func (f *InputField) Backspace() (deleted rune) {
	if f.AtBOL() {
		return 0
	}
	f.AdvanceColumn(-1)
	return f.DeleteRune()
}

func (f *InputField) KillToEnd() (deleted []rune) {
	if f.AtEOL() {
		return nil
	}
	deleted = slices.Clone(f.runes[f.point:])
	f.runes = f.runes[:f.point]
	return deleted
}

func (f *InputField) OnChar(char rune) {
	if char == 0 || char < 32 {
		return
	}
	f.InsertRune(char)
}

func (f *InputField) Reset() {
	f.runes = nil
	f.point = 0
	f.left = 0
}

func (f *InputField) ensureCursorVisible(width int) {
	if f.point < f.left {
		f.left = f.point
	}
	if f.point >= f.left+width {
		f.left = f.point - width + 1
	}
	if f.left < 0 {
		f.left = 0
	}
	if f.left > f.point {
		f.left = f.point
	}
}

func (f *InputField) Render(tp TilePane) {
	width := tp.Width()
	height := tp.Height()
	if width <= 0 || height <= 0 {
		return
	}

	f.ensureCursorVisible(width)

	for x := range width {
		idx := f.left + x
		r := ' '
		if idx < len(f.runes) {
			r = f.runes[idx]
		}
		if idx == f.point {
			tp.WithBg(ColorHighlight, func() {
				tp.DrawRune(x, 0, r)
			})
		} else {
			tp.DrawRune(x, 0, r)
		}
	}
}
