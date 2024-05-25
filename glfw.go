package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
	"github.com/go-gl/glfw/v3.3/glfw"
	"runtime"
)

const desiredFPS = 30

var fbSize Size

func init() {
	runtime.LockOSThread()
}

func GetTime() float64 {
	return glfw.GetTime()
}

type GlfwApp interface {
	Init() error
	IsRunning() bool
	OnKey(key glfw.Key, scancode int, action glfw.Action, modes glfw.ModifierKey)
	OnChar(char rune)
	OnFramebufferSize(width, height int)
	Render() error
	Update() error
	Close() error
}

func WithGL(windowTitle string, app GlfwApp) error {
	err := glfw.Init()
	if err != nil {
		return err
	}
	defer glfw.Terminate()

	monitor := glfw.GetPrimaryMonitor()
	if monitor == nil {
		return fmt.Errorf("no monitors found")
	}
	mode := monitor.GetVideoMode()
	if mode == nil {
		return fmt.Errorf("video mode cannot be determined")
	}
	glfw.WindowHint(glfw.RedBits, mode.RedBits)
	glfw.WindowHint(glfw.GreenBits, mode.GreenBits)
	glfw.WindowHint(glfw.BlueBits, mode.BlueBits)
	glfw.WindowHint(glfw.RefreshRate, mode.RefreshRate)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Focused, glfw.True)
	glfw.WindowHint(glfw.AutoIconify, glfw.False)
	glfw.WindowHint(glfw.DoubleBuffer, glfw.True)
	glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	window, err := glfw.CreateWindow(mode.Width, mode.Height, windowTitle, monitor, nil)
	if err != nil {
		return err
	}
	defer window.Destroy()
	framebufferSizeCallback := func(w *glfw.Window, width, height int) {
		fbSize.X = width
		fbSize.Y = height
		gl.Viewport(0, 0, int32(width), int32(height))
		app.OnFramebufferSize(width, height)
	}
	window.SetFramebufferSizeCallback(framebufferSizeCallback)
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		app.OnKey(key, scancode, action, mods)
	})
	window.SetCharCallback(func(w *glfw.Window, char rune) {
		app.OnChar(char)
	})
	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		return err
	}
	width, height := glfw.GetCurrentContext().GetFramebufferSize()
	framebufferSizeCallback(nil, width, height)
	if err := app.Init(); err != nil {
		return err
	}
	defer app.Close()
	for app.IsRunning() {
		start := glfw.GetTime()
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		if err := app.Render(); err != nil {
			return err
		}
		window.SwapBuffers()
		elapsedSeconds := glfw.GetTime() - start
		frameSeconds := 1.0 / desiredFPS
		if frameSeconds > elapsedSeconds {
			glfw.WaitEventsTimeout(frameSeconds - elapsedSeconds)
		} else {
			glfw.PollEvents()
		}
		if err := app.Update(); err != nil {
			return err
		}
	}
	return nil
}
