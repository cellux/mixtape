package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
	mgl "github.com/go-gl/mathgl/mgl32"
	"image"
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
    uniform mat4 u_transform;
    varying vec2 v_texcoord;
    void main(void) {
      gl_Position = u_transform * vec4(a_position, 0.0, 1.0);
      v_texcoord = a_texcoord;
    };` + "\x00"
	tileFragmentShaderA = `
    precision highp float;
    uniform sampler2D u_tex;
    varying vec2 v_texcoord;
    void main(void) {
      gl_FragColor = vec4(texture2D(u_tex, v_texcoord).a);
    };` + "\x00"
	tileFragmentShaderRGBA = `
    precision highp float;
    uniform sampler2D u_tex;
    varying vec2 v_texcoord;
    void main(void) {
      gl_FragColor = texture2D(u_tex, v_texcoord);
    };` + "\x00"
)

type TileVertex struct {
	position [2]float32
	texcoord [2]float32
}

type TileScreen struct {
	tm          *TileMap
	vertices    []TileVertex
	program     Program
	a_position  int32
	a_texcoord  int32
	u_transform int32
	u_tex       int32
}

func (tm *TileMap) CreateTileScreen() (*TileScreen, error) {
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
		u_transform: program.GetUniformLocation("u_transform\x00"),
		u_tex:       program.GetUniformLocation("u_tex\x00"),
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
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y1},
		texcoord: [2]float32{s0, t1},
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x1, y0},
		texcoord: [2]float32{s1, t0},
	})
	ts.vertices = append(ts.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
	})
}

func (ts *TileScreen) DrawString(x, y int, s string) {
	for offset, r := range s {
		ts.DrawRune(x+offset, y, r)
	}
}

func (ts *TileScreen) Render() error {
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
	tileSize := tm.GetTileSize()
	rectSizeInTiles := Size{
		X: fbSize.X / tileSize.X,
		Y: fbSize.Y / tileSize.Y,
	}
	borderSize := Size{
		X: (fbSize.X % tileSize.X) / 2,
		Y: (fbSize.Y % tileSize.Y) / 2,
	}
	rect := Rect{
		Min: Point{borderSize.X, borderSize.Y},
		Max: Point{fbSize.X - borderSize.X, fbSize.Y - borderSize.Y},
	}
	ux := 2.0 / float32(fbSize.X)
	uy := 2.0 / float32(fbSize.Y)
	wx := ux * float32(rect.Size().X)
	wy := uy * float32(rect.Size().Y)
	sx := wx / float32(rectSizeInTiles.X)
	sy := wy / float32(rectSizeInTiles.Y)
	mScale := mgl.Scale3D(sx, sy, 1)
	tx := -1.0 + ux*float32(rect.Min.X)
	ty := 1.0 - uy*float32(rect.Min.Y)
	mTranslate := mgl.Translate3D(tx, ty, 0)
	mTransform := mTranslate.Mul4(mScale)
	gl.UniformMatrix4fv(ts.u_transform, 1, false, &mTransform[0])
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(ts.vertices)))
	gl.Disable(gl.BLEND)
	gl.DisableVertexAttribArray(uint32(ts.a_position))
	gl.DisableVertexAttribArray(uint32(ts.a_texcoord))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return nil
}

func (ts *TileScreen) Close() error {
	return ts.program.Close()
}
