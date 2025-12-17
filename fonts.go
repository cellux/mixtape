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

type faceKey struct {
	size  FontSizeInPoints
	scale float32
}

type Font struct {
	font  *opentype.Font
	faces map[faceKey]font.Face
}

func (f *Font) GetFace(size FontSizeInPoints, scale float32) (font.Face, error) {
	key := faceKey{size: size, scale: scale}
	if face, ok := f.faces[key]; ok {
		return face, nil
	}
	// Use the content scale to compute the effective DPI. A base DPI of 96
	// is assumed for 1x scale; HiDPI displays (scale > 1) render at
	// proportionally higher resolution for crisp glyphs.
	dpi := 96.0 * float64(scale)
	faceOpts := &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	}
	face, err := opentype.NewFace(f.font, faceOpts)
	if err != nil {
		return nil, err
	}
	f.faces[key] = face
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

		// Clip to the tile's cell. Some glyphs/fonts can extend outside the expected
		// cell bounds (e.g. negative bearings), which would otherwise scribble into
		// neighboring glyph cells in the atlas.
		cellRect := image.Rect(col*maxWidth, row*tileHeight, (col+1)*maxWidth, (row+1)*tileHeight)
		clipped := dstRect.Intersect(cellRect)
		if clipped.Empty() {
			continue
		}
		dx := clipped.Min.X - dstRect.Min.X
		dy := clipped.Min.Y - dstRect.Min.Y
		maskPt = image.Point{X: maskPt.X + dx, Y: maskPt.Y + dy}

		draw.Draw(atlas, clipped, mask, maskPt, draw.Src)
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
		faces: make(map[faceKey]font.Face),
	}, nil
}

func LoadFontFromFile(name string) (*Font, error) {
	bytes, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return LoadFontFromBytes(bytes)
}
