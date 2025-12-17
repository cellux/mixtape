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
		Hinting: font.HintingFull,
	}
	face, err := opentype.NewFace(f.font, faceOpts)
	if err != nil {
		return nil, err
	}
	f.faces[size] = face
	return face, nil
}

func (f *Font) GetFaceImage(face font.Face, sizeInTiles Size) (image.Image, error) {
	cols, rows := sizeInTiles.X, sizeInTiles.Y
	if cols <= 0 || rows <= 0 {
		return nil, fmt.Errorf("sizeInTiles must be positive, got %v", sizeInTiles)
	}
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	tileHeight := metrics.Height.Ceil()
	if tileHeight == 0 {
		tileHeight = ascent + descent
	}
	nGlyphs := cols * rows
	maxWidth := 0
	for i := range nGlyphs {
		r := rune(i)
		if adv, ok := face.GlyphAdvance(r); ok {
			if w := adv.Ceil(); w > maxWidth {
				maxWidth = w
			}
		}
	}
	if maxWidth <= 0 {
		adv, ok := face.GlyphAdvance('m')
		if !ok {
			return nil, fmt.Errorf("Font face does not provide a glyph for rune 'm'")
		}
		maxWidth = adv.Ceil()
	}

	atlas := image.NewAlpha(image.Rect(0, 0, maxWidth*cols, tileHeight*rows))
	for i := range nGlyphs {
		r := rune(i)
		col := i % cols
		row := i / cols
		dot := fixed.Point26_6{
			X: fixed.I(col * maxWidth),
			Y: fixed.I(row*tileHeight + ascent),
		}
		dstRect, mask, maskPt, _, ok := face.Glyph(dot, r)
		if !ok || mask == nil {
			continue
		}
		draw.Draw(atlas, dstRect, mask, maskPt, draw.Src)
	}
	return atlas, nil
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
