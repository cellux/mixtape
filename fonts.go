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

type Font struct {
	font  *opentype.Font
	faces map[float64]font.Face
}

func (f *Font) GetFace(size float64) (font.Face, error) {
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

func (f *Font) GetFaceImage(face font.Face, cols, rows int) (image.Image, error) {
	metrics := face.Metrics()
	descent := metrics.Descent.Ceil()
	ascent := metrics.Ascent.Ceil()
	height := descent + ascent
	mAdvance, ok := face.GlyphAdvance('m')
	if !ok {
		return nil, fmt.Errorf("Font face does not provide a glyph for rune 'm'")
	}
	width := mAdvance.Ceil()
	result := image.NewAlpha(image.Rect(0, 0, width*cols, height*rows))
	d := font.Drawer{
		Src:  image.Opaque,
		Face: face,
	}
	cellRect := image.Rect(0, 0, width, height)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			d.Dst = image.NewAlpha(cellRect)
			d.Dot = fixed.P(0, height-descent)
			d.DrawString(string(rune(y*cols + x)))
			dp := image.Pt(x*width, y*height)
			draw.Copy(result, dp, d.Dst, cellRect, draw.Src, nil)
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
		faces: make(map[float64]font.Face),
	}, nil
}

func LoadFontFromFile(name string) (*Font, error) {
	bytes, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return LoadFontFromBytes(bytes)
}
