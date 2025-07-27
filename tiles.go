package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
	mgl "github.com/go-gl/mathgl/mgl32"
	"image"
	"math"
	"unsafe"
)

type TileMap struct {
	img        image.Image
	cols, rows int
	tex        Texture
}

func CreateTileMap(img image.Image, cols, rows int) (*TileMap, error) {
	tex, err := CreateTexture()
	if err != nil {
		return nil, err
	}
	mapSize := img.Bounds().Size()
	switch img := img.(type) {
	case *image.Alpha:
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA,
			int32(mapSize.X), int32(mapSize.Y),
			0, gl.ALPHA, gl.UNSIGNED_BYTE,
			gl.Ptr(img.Pix))
	case *image.RGBA:
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA,
			int32(mapSize.X), int32(mapSize.Y),
			0, gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(img.Pix))
	default:
		return nil, fmt.Errorf("cannot create TileMap OpenGL texture from image of type %T", img)
	}
	tm := &TileMap{
		img:  img,
		cols: cols,
		rows: rows,
		tex:  tex,
	}
	return tm, nil
}

func (tm *TileMap) GetMapSize() Size {
	return tm.img.Bounds().Size()
}

func (tm *TileMap) GetTileSize() Size {
	mapSize := tm.GetMapSize()
	return Size{X: mapSize.X / tm.cols, Y: mapSize.Y / tm.rows}
}

func (tm *TileMap) Close() error {
	return tm.tex.Close()
}

const (
	tileVertexShader = `
    precision highp float;
    attribute vec2 a_position;
    attribute vec2 a_texcoord;
    attribute vec4 a_fgColor;
    attribute vec4 a_bgColor;
    uniform mat4 u_transform;
    varying vec2 v_texcoord;
    varying vec4 v_fgColor;
    varying vec4 v_bgColor;
    void main(void) {
      gl_Position = u_transform * vec4(a_position, 0.0, 1.0);
      v_texcoord = a_texcoord;
      v_fgColor = a_fgColor;
      v_bgColor = a_bgColor;
    };` + "\x00"
	tileFragmentShaderA = `
    precision highp float;
    uniform sampler2D u_tex;
    varying vec2 v_texcoord;
    varying vec4 v_fgColor;
    varying vec4 v_bgColor;
    void main(void) {
      vec4 t = texture2D(u_tex, v_texcoord);
      gl_FragColor = v_bgColor + v_fgColor * t.a;
    };` + "\x00"
	tileFragmentShaderRGBA = `
    precision highp float;
    uniform sampler2D u_tex;
    varying vec2 v_texcoord;
    varying vec4 v_fgColor;
    varying vec4 v_bgColor;
    void main(void) {
      vec4 t = texture2D(u_tex, v_texcoord);
      gl_FragColor = v_bgColor + v_fgColor * t;
    };` + "\x00"
)

type TileVertex struct {
	position [2]float32
	texcoord [2]float32
	fgColor  [4]float32
	bgColor  [4]float32
}

type TileScreen struct {
	tm          *TileMap
	vertices    []TileVertex
	program     Program
	a_position  int32
	a_texcoord  int32
	a_fgColor   int32
	a_bgColor   int32
	u_transform int32
	u_tex       int32
	fgColor     Color
	bgColor     Color
}

func (tm *TileMap) CreateScreen() (*TileScreen, error) {
	program, err := func() (Program, error) {
		switch img := tm.img.(type) {
		case *image.Alpha:
			return CreateProgram(tileVertexShader, tileFragmentShaderA)
		case *image.RGBA:
			return CreateProgram(tileVertexShader, tileFragmentShaderRGBA)
		default:
			return Program{}, fmt.Errorf("cannot create TileMap from image of type %T", img)
		}
	}()
	if err != nil {
		return nil, err
	}
	ts := &TileScreen{
		tm:          tm,
		vertices:    make([]TileVertex, 0, 6*4096),
		program:     program,
		a_position:  program.GetAttribLocation("a_position\x00"),
		a_texcoord:  program.GetAttribLocation("a_texcoord\x00"),
		a_fgColor:   program.GetAttribLocation("a_fgColor\x00"),
		a_bgColor:   program.GetAttribLocation("a_bgColor\x00"),
		u_transform: program.GetUniformLocation("u_transform\x00"),
		u_tex:       program.GetUniformLocation("u_tex\x00"),
		fgColor:     ColorWhite,
		bgColor:     ColorBlack,
	}
	return ts, nil
}

func (ts *TileScreen) Clear() {
	ts.vertices = ts.vertices[:0]
}

func (ts *TileScreen) DrawRune(x, y int, r rune) {
	rows := ts.tm.rows
	cols := ts.tm.cols
	col := int(r) % cols
	row := int(r) / cols
	x0 := float32(x)
	x1 := float32(x + 1)
	y0 := float32(-y)
	y1 := float32(-y - 1)
	tx := float32(1.0) / float32(cols)
	ty := float32(1.0) / float32(rows)
	s0 := float32(col) / float32(cols)
	s1 := float32(s0 + tx)
	t0 := float32(row) / float32(rows)
	t1 := float32(t0 + ty)
	fgColor := ColorTo4Float32(ts.fgColor)
	bgColor := ColorTo4Float32(ts.bgColor)
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y1},
		texcoord: [2]float32{s0, t1},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y0},
		texcoord: [2]float32{s1, t0},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
		fgColor:  fgColor,
		bgColor:  bgColor,
	})
}

func (ts *TileScreen) SetFg(c Color) {
	ts.fgColor = c
}

func (ts *TileScreen) SetBg(c Color) {
	ts.bgColor = c
}

func (ts *TileScreen) DrawString(x, y int, s string) {
	for offset, r := range s {
		ts.DrawRune(x+offset, y, r)
	}
}

func (ts *TileScreen) Render() {
	if len(ts.vertices) == 0 {
		return
	}
	tm := ts.tm
	ts.program.Use()
	tm.tex.Bind()
	var activeTexture int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &activeTexture)
	gl.Uniform1i(ts.u_tex, activeTexture-gl.TEXTURE0)
	gl.EnableVertexAttribArray(uint32(ts.a_position))
	gl.VertexAttribPointer(
		uint32(ts.a_position), 2, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&ts.vertices[0].position[0]))
	gl.EnableVertexAttribArray(uint32(ts.a_texcoord))
	gl.VertexAttribPointer(
		uint32(ts.a_texcoord), 2, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&ts.vertices[0].texcoord[0]))
	gl.EnableVertexAttribArray(uint32(ts.a_fgColor))
	gl.VertexAttribPointer(
		uint32(ts.a_fgColor), 3, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&ts.vertices[0].fgColor[0]))
	gl.EnableVertexAttribArray(uint32(ts.a_bgColor))
	gl.VertexAttribPointer(
		uint32(ts.a_bgColor), 3, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&ts.vertices[0].bgColor[0]))
	tileSize := tm.GetTileSize()
	rectSizeInTiles := Size{
		X: fbSize.X / tileSize.X,
		Y: fbSize.Y / tileSize.Y,
	}
	borderSize := Size{
		X: (fbSize.X % tileSize.X) / 2,
		Y: (fbSize.Y % tileSize.Y) / 2,
	}
	pixelRect := Rect{
		Min: Point{X: borderSize.X, Y: borderSize.Y},
		Max: Point{X: fbSize.X - borderSize.X, Y: fbSize.Y - borderSize.Y},
	}
	ux := 2.0 / float32(fbSize.X)
	uy := 2.0 / float32(fbSize.Y)
	wx := ux * float32(pixelRect.Size().X)
	wy := uy * float32(pixelRect.Size().Y)
	sx := wx / float32(rectSizeInTiles.X)
	sy := wy / float32(rectSizeInTiles.Y)
	mScale := mgl.Scale3D(sx, sy, 1)
	tx := -1.0 + ux*float32(pixelRect.Min.X)
	ty := 1.0 - uy*float32(pixelRect.Min.Y)
	mTranslate := mgl.Translate3D(tx, ty, 0)
	mTransform := mTranslate.Mul4(mScale)
	gl.UniformMatrix4fv(ts.u_transform, 1, false, &mTransform[0])
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(ts.vertices)))
	gl.Disable(gl.BLEND)
	gl.DisableVertexAttribArray(uint32(ts.a_position))
	gl.DisableVertexAttribArray(uint32(ts.a_texcoord))
	gl.DisableVertexAttribArray(uint32(ts.a_fgColor))
	gl.DisableVertexAttribArray(uint32(ts.a_bgColor))
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

type TilePane struct {
	ts   *TileScreen
	rect Rect
}

func (tp TilePane) Width() int {
	return tp.rect.Dx()
}

func (tp TilePane) Height() int {
	return tp.rect.Dy()
}

func (tp TilePane) GetPixelRect() Rect {
	tileSize := tp.ts.tm.GetTileSize()
	borderSize := Size{
		X: (fbSize.X % tileSize.X) / 2,
		Y: (fbSize.Y % tileSize.Y) / 2,
	}
	pixelRect := Rect{
		Min: Point{
			X: borderSize.X + tp.rect.Min.X*tileSize.X,
			Y: borderSize.Y + tp.rect.Min.Y*tileSize.Y,
		},
		Max: Point{
			X: borderSize.X + tp.rect.Max.X*tileSize.X,
			Y: borderSize.Y + tp.rect.Max.Y*tileSize.Y,
		},
	}
	return pixelRect
}

func (tp TilePane) SplitX(at float64) (TilePane, TilePane) {
	width := float64(tp.Width())
	if at < 0.0 {
		if at > -1.0 {
			at = 1.0 + at
		} else {
			at = width + at
		}
	}
	if at < 1.0 {
		at = math.Round(width * at)
	}
	if at > width {
		at = width
	}
	left := TilePane{
		ts: tp.ts,
		rect: Rect{
			Min: tp.rect.Min,
			Max: Point{X: tp.rect.Min.X + int(at), Y: tp.rect.Max.Y},
		},
	}
	right := TilePane{
		ts: tp.ts,
		rect: Rect{
			Min: Point{X: tp.rect.Min.X + int(at), Y: tp.rect.Min.Y},
			Max: tp.rect.Max,
		},
	}
	return left, right
}

func (tp TilePane) SplitY(at float64) (TilePane, TilePane) {
	height := float64(tp.Height())
	if at < 0.0 {
		if at > -1.0 {
			at = 1.0 + at
		} else {
			at = height + at
		}
	}
	if at < 1.0 {
		at = math.Round(height * at)
	}
	if at > height {
		at = height
	}
	top := TilePane{
		ts: tp.ts,
		rect: Rect{
			Min: tp.rect.Min,
			Max: Point{X: tp.rect.Max.X, Y: tp.rect.Min.Y + int(at)},
		},
	}
	bottom := TilePane{
		ts: tp.ts,
		rect: Rect{
			Min: Point{X: tp.rect.Min.X, Y: tp.rect.Min.Y + int(at)},
			Max: tp.rect.Max,
		},
	}
	return top, bottom
}

func (tp TilePane) SetFg(c Color) {
	tp.ts.SetFg(c)
}

func (tp TilePane) SetBg(c Color) {
	tp.ts.SetBg(c)
}

func (tp TilePane) WithFg(fg Color, fn func()) {
	defer tp.SetFg(tp.ts.fgColor)
	tp.SetFg(fg)
	fn()
}

func (tp TilePane) WithBg(bg Color, fn func()) {
	defer tp.SetBg(tp.ts.bgColor)
	tp.SetBg(bg)
	fn()
}

func (tp TilePane) WithFgBg(fg, bg Color, fn func()) {
	defer tp.SetFg(tp.ts.bgColor)
	defer tp.SetBg(tp.ts.bgColor)
	tp.SetFg(fg)
	tp.SetBg(bg)
	fn()
}

func (tp TilePane) DrawRune(x, y int, r rune) {
	rect := tp.rect
	screenX := rect.Min.X + x
	screenY := rect.Min.Y + y
	if screenX < rect.Max.X && screenY < rect.Max.Y {
		tp.ts.DrawRune(screenX, screenY, r)
	}
}

func (tp TilePane) DrawString(x, y int, s string) {
	for offset, r := range s {
		tp.DrawRune(x+offset, y, r)
	}
}

func (ts *TileScreen) GetPane() TilePane {
	tileSize := ts.tm.GetTileSize()
	return TilePane{
		ts: ts,
		rect: Rect{
			Min: Point{X: 0, Y: 0},
			Max: Point{X: fbSize.X / tileSize.X, Y: fbSize.Y / tileSize.Y},
		},
	}
}

func (ts *TileScreen) Close() error {
	return ts.program.Close()
}
