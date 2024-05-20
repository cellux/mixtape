package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
	mgl "github.com/go-gl/mathgl/mgl32"
	"image"
	"unsafe"
)

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
	tileFragmentShader = `
    precision highp float;
    uniform sampler2D u_tex;
    varying vec2 v_texcoord;
    void main(void) {
      gl_FragColor = vec4(texture2D(u_tex, v_texcoord).a);
    };` + "\x00"
)

type TileVertex struct {
	position [2]float32
	texcoord [2]float32
}

type TileMap struct {
	img         image.Image
	cols, rows  int
	tex         Texture
	program     Program
	a_position  int32
	a_texcoord  int32
	u_transform int32
	u_tex       int32
}

type TileDrawList struct {
	tm       *TileMap
	vertices []TileVertex
}

func CreateTileMap(img image.Image, cols, rows int) (*TileMap, error) {
	program, err := CreateProgram(tileVertexShader, tileFragmentShader)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("unsupported image format")
	}
	tm := &TileMap{
		img:         img,
		cols:        cols,
		rows:        rows,
		tex:         tex,
		program:     program,
		a_position:  program.GetAttribLocation("a_position\x00"),
		a_texcoord:  program.GetAttribLocation("a_texcoord\x00"),
		u_transform: program.GetUniformLocation("u_transform\x00"),
		u_tex:       program.GetUniformLocation("u_tex\x00"),
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

func (tm *TileMap) CreateDrawList() *TileDrawList {
	return &TileDrawList{
		tm:       tm,
		vertices: make([]TileVertex, 0, 6*4096),
	}
}

func (tdl *TileDrawList) Clear() {
	tdl.vertices = tdl.vertices[:0]
}

func (tdl *TileDrawList) DrawRune(x, y int, r rune) {
	rows := tdl.tm.rows
	cols := tdl.tm.cols
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
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
	})
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x0, y1},
		texcoord: [2]float32{s0, t1},
	})
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
	})
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x1, y1},
		texcoord: [2]float32{s1, t1},
	})
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x1, y0},
		texcoord: [2]float32{s1, t0},
	})
	tdl.vertices = append(tdl.vertices, TileVertex{
		position: [2]float32{x0, y0},
		texcoord: [2]float32{s0, t0},
	})
}

func (tdl *TileDrawList) DrawString(x, y int, s string) {
	for offset, r := range s {
		tdl.DrawRune(x+offset, y, r)
	}
}

func (tdl *TileDrawList) Render(rect Rect) error {
	tm := tdl.tm
	tm.program.Use()
	tm.tex.Bind()
	var activeTexture int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &activeTexture)
	gl.Uniform1i(tm.u_tex, activeTexture-gl.TEXTURE0)
	gl.EnableVertexAttribArray(uint32(tm.a_position))
	gl.VertexAttribPointer(
		uint32(tm.a_position), 2, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&tdl.vertices[0].position[0]))
	gl.EnableVertexAttribArray(uint32(tm.a_texcoord))
	gl.VertexAttribPointer(
		uint32(tm.a_texcoord), 2, gl.FLOAT, false,
		int32(unsafe.Sizeof(TileVertex{})),
		gl.Ptr(&tdl.vertices[0].texcoord[0]))
	tileSize := tm.GetTileSize()
	rectSizeInTiles := Size{
		X: rect.Size().X / tileSize.X,
		Y: rect.Size().Y / tileSize.Y,
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
	gl.UniformMatrix4fv(tm.u_transform, 1, false, &mTransform[0])
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(tdl.vertices)))
	gl.Disable(gl.BLEND)
	gl.DisableVertexAttribArray(uint32(tm.a_position))
	gl.DisableVertexAttribArray(uint32(tm.a_texcoord))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return nil
}

func (tm *TileMap) Close() error {
	tm.program.Close()
	return nil
}
