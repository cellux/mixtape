package main

import (
	"image"
)

type Point = image.Point
type Size = image.Point
type Rect = image.Rectangle

type Smp = float64

type SmpUnOp = func(x Smp) Smp
type SmpBinOp = func(x, y Smp) Smp

type Frame = []Smp

// Screen is a UI screen that can render itself and provide a keymap overlay.
type Screen interface {
	Render(app *App, ts *TileScreen)
	HandleKey(key Key) (KeyHandler, bool)
	Reset()
	Close()
}

// CharScreen is implemented by screens that want to handle character input.
type CharScreen interface {
	OnChar(app *App, char rune)
}
