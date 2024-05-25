package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
)

type Texture struct {
	tex uint32
}

func (t Texture) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, t.tex)
}

func CreateTexture() (Texture, error) {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	return Texture{tex}, nil
}

func (t Texture) Close() error {
	if t.tex != 0 {
		gl.DeleteTextures(1, &t.tex)
		t.tex = 0
	}
	return nil
}

type Shader struct {
	shader uint32
}

func GetShaderInfoLog(shader uint32) string {
	var length int32
	gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &length)
	log := make([]uint8, length)
	var logLen int32
	gl.GetShaderInfoLog(shader, length, &logLen, &log[0])
	return string(log[:logLen])
}

func CreateShader(shaderType uint32, source string) (Shader, error) {
	shader := gl.CreateShader(shaderType)
	data := gl.Str(source)
	length := int32(len(source))
	gl.ShaderSource(shader, 1, &data, &length)
	gl.CompileShader(shader)
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		return Shader{}, fmt.Errorf("shader compilation failed: %s", GetShaderInfoLog(shader))
	}
	return Shader{shader}, nil
}

func (s Shader) Close() error {
	if s.shader != 0 {
		gl.DeleteShader(s.shader)
		s.shader = 0
	}
	return nil
}

type Program struct {
	program        uint32
	vertexShader   Shader
	fragmentShader Shader
}

func GetProgramInfoLog(program uint32) string {
	var length int32
	gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &length)
	log := make([]uint8, length)
	var logLen int32
	gl.GetProgramInfoLog(program, length, &logLen, &log[0])
	return string(log[:logLen])
}

func CreateProgram(vertexShader string, fragmentShader string) (Program, error) {
	vs, err := CreateShader(gl.VERTEX_SHADER, vertexShader)
	if err != nil {
		return Program{}, err
	}
	fs, err := CreateShader(gl.FRAGMENT_SHADER, fragmentShader)
	if err != nil {
		return Program{}, err
	}
	program := gl.CreateProgram()
	gl.AttachShader(program, vs.shader)
	gl.AttachShader(program, fs.shader)
	gl.LinkProgram(program)
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		return Program{}, fmt.Errorf("program link failed: %s", GetProgramInfoLog(program))
	}
	return Program{program, vs, fs}, nil
}

func (p Program) GetAttribLocation(name string) int32 {
	return gl.GetAttribLocation(p.program, gl.Str(name))
}

func (p Program) GetUniformLocation(name string) int32 {
	return gl.GetUniformLocation(p.program, gl.Str(name))
}

func (p Program) Use() {
	gl.UseProgram(p.program)
}

func (p Program) Close() error {
	if err := p.vertexShader.Close(); err != nil {
		return err
	}
	if err := p.fragmentShader.Close(); err != nil {
		return err
	}
	if p.program != 0 {
		gl.DeleteProgram(p.program)
		p.program = 0
	}
	return nil
}
