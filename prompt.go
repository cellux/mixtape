package main

import "fmt"

type PromptInputMode int

const (
	PromptInputModeText PromptInputMode = iota
	PromptInputModeChar
)

type PromptCallbacks struct {
	onConfirm func(string)
	onCancel  func()
}

type Prompt struct {
	mode      PromptInputMode
	prompt    string
	chars     []rune
	keymap    KeyMap
	input     *InputField
	callbacks PromptCallbacks
}

func CreateTextPrompt(prompt string, callbacks PromptCallbacks) *Prompt {
	p := &Prompt{
		mode:      PromptInputModeText,
		prompt:    prompt,
		callbacks: callbacks,
	}
	p.input = CreateInputField(InputFieldCallbacks{
		onConfirm: p.handleTextConfirm,
		onCancel:  p.handleCancel,
	})
	return p
}

func CreateCharPrompt(prompt string, chars string, callbacks PromptCallbacks) *Prompt {
	p := &Prompt{
		mode:      PromptInputModeChar,
		prompt:    prompt,
		chars:     []rune(chars),
		callbacks: callbacks,
	}
	p.initKeymap()
	return p
}

func (p *Prompt) SetText(text string) {
	if p.mode != PromptInputModeText {
		return
	}
	p.input.SetText(text)
}

func (p *Prompt) Text() string {
	if p.mode != PromptInputModeText {
		return ""
	}
	return p.input.Text()
}

func (p *Prompt) Reset() {
	switch p.mode {
	case PromptInputModeText:
		p.input.Reset()
	case PromptInputModeChar:
		// nothing to reset
	}
}

func (p *Prompt) HandleKey(key Key) (KeyHandler, bool) {
	switch p.mode {
	case PromptInputModeText:
		return p.input.HandleKey(key)
	case PromptInputModeChar:
		return p.keymap.HandleKey(key)
	default:
		return nil, false
	}
}

func (p *Prompt) OnChar(char rune) {
	switch p.mode {
	case PromptInputModeText:
		p.input.OnChar(char)
	case PromptInputModeChar:
		if p.isAllowedChar(char) {
			p.handleCharConfirm(char)
		}
	}
}

func (p *Prompt) Render(tp TilePane) {
	width := tp.Width()
	height := tp.Height()
	if width <= 0 || height <= 0 {
		return
	}

	linePane := tp.SubPane(0, height-1, width, 1)
	switch p.mode {
	case PromptInputModeText:
		linePane.DrawString(0, 0, p.prompt)
		inputPane := linePane.SubPane(len(p.prompt), 0, width-len(p.prompt), 1)
		p.input.Render(inputPane)
	case PromptInputModeChar:
		linePane.DrawString(0, 0, fmt.Sprintf("%s [%s]", p.prompt, string(p.chars)))
	}
}

func (p *Prompt) initKeymap() {
	p.keymap = CreateKeyMap()
	p.keymap.Bind("Escape", p.handleCancel)
	p.keymap.Bind("C-g", p.handleCancel)
}

func (p *Prompt) handleTextConfirm() {
	if p.callbacks.onConfirm != nil {
		p.callbacks.onConfirm(p.input.Text())
	}
}

func (p *Prompt) handleCharConfirm(char rune) {
	if p.callbacks.onConfirm != nil {
		p.callbacks.onConfirm(string(char))
	}
}

func (p *Prompt) handleCancel() {
	if p.callbacks.onCancel != nil {
		p.callbacks.onCancel()
	}
}

func (p *Prompt) isAllowedChar(char rune) bool {
	for _, c := range p.chars {
		if c == char {
			return true
		}
	}
	return false
}
