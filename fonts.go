package main

import (
	"fmt"
	"image"
	"os"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type FontSizeInPoints = float64

type Font struct {
	font  *opentype.Font
	faces map[FontSizeInPoints]font.Face
}

func (f *Font) GetFace(size FontSizeInPoints) (font.Face, error) {
	if face, ok := f.faces[size]; ok {
		return face, nil
	}
	faceOpts := &opentype.FaceOptions{
		Size:    size,
		DPI:     96,
		Hinting: font.HintingNone,
	}
	return opentype.NewFace(f.font, faceOpts)
}

func (f *Font) GetFaceImage(face font.Face, sizeInTiles Size) (image.Image, error) {
	metrics := face.Metrics()
	descent := metrics.Descent.Ceil()
	ascent := metrics.Ascent.Ceil()
	charHeight := descent + ascent
	mAdvance, ok := face.GlyphAdvance('m')
	if !ok {
		return nil, fmt.Errorf("Font face does not provide a glyph for rune 'm'")
	}
	charWidth := mAdvance.Ceil()
	cols, rows := sizeInTiles.X, sizeInTiles.Y
	result := image.NewAlpha(image.Rect(0, 0, charWidth*cols, charHeight*rows))
	d := font.Drawer{
		Src:  image.Opaque,
		Face: face,
	}
	charRect := image.Rect(0, 0, charWidth, charHeight)
	for y := range rows {
		for x := range cols {
			d.Dst = image.NewAlpha(charRect)
			d.Dot = fixed.P(0, charHeight-descent)
			d.DrawString(string(rune(y*cols + x)))
			dp := image.Pt(x*charWidth, y*charHeight)
			draw.Copy(result, dp, d.Dst, charRect, draw.Src, nil)
		}
	}
	return result, nil
}

func LoadFontFromBytes(bytes []byte) (*Font, error) {
	f, err := opentype.Parse(bytes)
	if err != nil {
		return nil, err
	}
	return &Font{
		font:  f,
		faces: make(map[FontSizeInPoints]font.Face),
	}, nil
}

func LoadFontFromFile(name string) (*Font, error) {
	bytes, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return LoadFontFromBytes(bytes)
}
