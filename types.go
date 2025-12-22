package main

import (
	"image"
)

type Point = image.Point
type Size = image.Point
type Rect = image.Rectangle

type Smp = float64

type SmpBinOp = func(x, y Smp) Smp

type Frame = []Smp
