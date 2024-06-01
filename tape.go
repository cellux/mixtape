package main

import (
	"fmt"
	gl "github.com/go-gl/gl/v3.1/gles2"
	mgl "github.com/go-gl/mathgl/mgl32"
	"math"
	"unsafe"
)

type Tape struct {
	nchannels int
	nframes   int
	samples   []Smp
}

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(nchannels=%d nframes=%d)", t.nchannels, t.nframes)
}

func (t *Tape) GetSampleIterator() SampleIterator {
	sampleIndex := 0
	end := t.nframes * t.nchannels
	if t.nchannels == 1 {
		return func() Smp {
			if sampleIndex < end {
				nextSample := t.samples[sampleIndex]
				sampleIndex++
				return nextSample
			} else {
				return 0
			}
		}
	} else {
		return func() Smp {
			if sampleIndex < end {
				var nextSample Smp
				for range t.nchannels {
					nextSample += t.samples[sampleIndex]
					sampleIndex++
				}
				return nextSample / float64(t.nchannels)
			} else {
				return 0
			}
		}
	}
}

func (t *Tape) GetInterpolatedSampleAt(channel int, frame float64) Smp {
	frameIndexLo := math.Floor(frame)
	sampleIndexLo := channel + int(frameIndexLo)*t.nchannels
	smpLo := t.samples[sampleIndexLo]
	smpHi := t.samples[sampleIndexLo+1]
	frameIndexDelta := frame - frameIndexLo
	return smpLo + (smpHi-smpLo)*frameIndexDelta
}

func clamp(value float64, lo float64, hi float64) float64 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

func calcSin(phase float64) float64 {
	return math.Sin(phase * 2 * math.Pi)
}

func calcPulse(phase, width float64) float64 {
	if phase < width {
		return -1.0
	} else {
		return 1.0
	}
}

func calcTriangle(phase float64) float64 {
	if phase < 0.25 {
		return phase * 4.0
	} else if phase < 0.75 {
		return 1.0 - (phase-0.25)*4.0
	} else {
		return -1.0 + (phase-0.75)*4.0
	}
}

func calcSaw(phase float64) float64 {
	if phase < 0.5 {
		return phase * 2.0
	} else {
		return -1.0 + (phase-0.5)*2.0
	}
}

func init() {
	RegisterMethod[*Tape]("pulse", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		width := clamp(vm.GetFloat(":width"), 0, 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcPulse(phase, width)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase += incr
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.pulse", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		sr := vm.GetFloat(":sr")
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		width := clamp(vm.GetFloat(":width"), 0, 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcPulse(phase, width)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("triangle", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcTriangle(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase += incr
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.triangle", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		sr := vm.GetFloat(":sr")
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcTriangle(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("saw", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSaw(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase += incr
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.saw", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		sr := vm.GetFloat(":sr")
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSaw(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("sin", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSin(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase += incr
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.sin", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		sr := vm.GetFloat(":sr")
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := clamp(vm.GetFloat(":phase"), 0, 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSin(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})
}

func pushTape(vm *VM, nchannels, nframes int) {
	samples := make([]Smp, nchannels*(nframes+1))
	tape := &Tape{
		nchannels: nchannels,
		nframes:   nframes,
		samples:   samples,
	}
	vm.PushVal(tape)
}

func init() {
	RegisterWord("tape1", func(vm *VM) error {
		nframes := int(Pop[Num](vm))
		pushTape(vm, 1, nframes)
		return nil
	})

	RegisterWord("tape2", func(vm *VM) error {
		nframes := int(Pop[Num](vm))
		pushTape(vm, 2, nframes)
		return nil
	})
}

const (
	pointVertexShader = `
    precision highp float;
    attribute vec2 a_position;
    uniform mat4 u_transform;
    void main(void) {
      gl_Position = u_transform * vec4(a_position, 0.0, 1.0);
    };` + "\x00"
	pointFragmentShader = `
    precision highp float;
    void main(void) {
      gl_FragColor = vec4(1.0);
    };` + "\x00"
)

type PointVertex struct {
	position [2]float32
}

type TapeDisplay struct {
	tape        *Tape
	pixelRect   Rect
	vertices    [][]PointVertex
	program     Program
	a_position  int32
	u_transform int32
}

func CreateTapeDisplay() (*TapeDisplay, error) {
	program, err := CreateProgram(pointVertexShader, pointFragmentShader)
	if err != nil {
		return nil, err
	}
	td := &TapeDisplay{
		program:     program,
		a_position:  program.GetAttribLocation("a_position\x00"),
		u_transform: program.GetUniformLocation("u_transform\x00"),
	}
	return td, nil
}

func (td *TapeDisplay) Render(tape *Tape, pixelRect Rect, windowSize int, windowOffset int) {
	pixelWidth, pixelHeight := pixelRect.Dx(), pixelRect.Dy()
	if td.tape != tape || td.pixelRect != pixelRect {
		td.tape = tape
		td.pixelRect = pixelRect
		td.vertices = make([][]PointVertex, tape.nchannels)
		for ch := range tape.nchannels {
			td.vertices[ch] = make([]PointVertex, pixelWidth)
			for x := 0; x < pixelWidth; x++ {
				td.vertices[ch][x].position[0] = float32(x) + 0.5
			}
		}
	}
	channelHeight := float32(pixelHeight) / float32(tape.nchannels)
	channelHeightHalf := channelHeight / 2.0
	incr := float64(windowSize) / float64(pixelWidth)
	readIndex := float64(windowOffset)
	for x := 0; x < pixelWidth; x++ {
		channelTop := float32(0)
		for ch := range tape.nchannels {
			smp := tape.GetInterpolatedSampleAt(ch, readIndex)
			td.vertices[ch][x].position[1] = channelTop + channelHeightHalf - float32(smp)*channelHeightHalf
			channelTop += channelHeight
		}
		readIndex += incr
	}
	td.program.Use()
	gl.EnableVertexAttribArray(uint32(td.a_position))
	ux := 2.0 / float32(fbSize.X)
	uy := 2.0 / float32(fbSize.Y)
	mScale := mgl.Scale3D(ux, -uy, 1)
	tx := -1.0 + ux*float32(pixelRect.Min.X)
	ty := 1.0 - uy*float32(pixelRect.Min.Y)
	mTranslate := mgl.Translate3D(tx, ty, 0)
	mTransform := mTranslate.Mul4(mScale)
	gl.UniformMatrix4fv(td.u_transform, 1, false, &mTransform[0])
	for ch := range tape.nchannels {
		gl.VertexAttribPointer(
			uint32(td.a_position), 2, gl.FLOAT, false,
			int32(unsafe.Sizeof(PointVertex{})),
			gl.Ptr(&td.vertices[ch][0].position[0]))
		gl.DrawArrays(gl.LINE_STRIP, 0, int32(len(td.vertices[ch])))
	}
	gl.DisableVertexAttribArray(uint32(td.a_position))
}
