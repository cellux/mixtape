package main

import (
	"fmt"
	"iter"
	"math"
)

const DefaultWavetableSize = 2048

// Wavetable represents a collection of single-cycle frames with optional frame morphing.
type Wavetable struct {
	frames    [][]Smp
	frameSize int
}

func (wt *Wavetable) getVal() Val { return wt }

func (wt *Wavetable) String() string {
	return fmt.Sprintf("Wavetable(%d frames, size %d)", len(wt.frames), wt.frameSize)
}

// sampleFrame returns a sample from a single frame at fractional phase [0,1).
func sampleFrame(frame []Smp, phase Smp) Smp {
	if len(frame) == 0 {
		return 0
	}
	// wrap phase and scale to frame size
	p := math.Mod(float64(phase), 1.0)
	if p < 0 {
		p += 1.0
	}
	pos := p * float64(len(frame))
	i0 := int(pos)
	frac := pos - float64(i0)
	i1 := (i0 + 1) % len(frame)
	return frame[i0]*(1.0-frac) + frame[i1]*frac
}

// Sample returns a sample at oscillator phase with frame-morph value in [0,1].
func (wt *Wavetable) Sample(phase, morph Smp) Smp {
	if wt == nil || len(wt.frames) == 0 || wt.frameSize == 0 {
		return 0
	}
	m := float64(morph)
	if m < 0 {
		m = 0
	}
	if m > 1 {
		m = 1
	}
	if len(wt.frames) == 1 {
		return sampleFrame(wt.frames[0], phase)
	}
	idx := m * float64(len(wt.frames)-1)
	i0 := int(idx)
	frac := idx - float64(i0)
	i1 := i0 + 1
	if i1 >= len(wt.frames) {
		i1 = len(wt.frames) - 1
		frac = 0
	}
	s0 := sampleFrame(wt.frames[i0], phase)
	s1 := sampleFrame(wt.frames[i1], phase)
	return Smp(float64(s0)*(1.0-frac) + float64(s1)*frac)
}

func newWavetableFromFrames(frames [][]Smp) (*Wavetable, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("wavetable: no frames")
	}
	size := len(frames[0])
	if size == 0 {
		return nil, fmt.Errorf("wavetable: empty frame")
	}
	for i, f := range frames {
		if len(f) != size {
			return nil, fmt.Errorf("wavetable: frame %d has size %d, expected %d", i, len(f), size)
		}
	}
	return &Wavetable{frames: frames, frameSize: size}, nil
}

func wavetableFromVec(v Vec) (*Wavetable, error) {
	// Treat a flat numeric vector as one frame.
	frame := make([]Smp, len(v))
	for i, item := range v {
		n, ok := item.(Num)
		if !ok {
			return nil, fmt.Errorf("wavetable: expected numeric vector, got %T at index %d", item, i)
		}
		frame[i] = Smp(n)
	}
	return newWavetableFromFrames([][]Smp{frame})
}

func frameFromTape(t *Tape) []Smp {
	frame := make([]Smp, t.nframes)
	// take first channel
	for i := 0; i < t.nframes; i++ {
		frame[i] = t.samples[i*t.nchannels]
	}
	return frame
}

func wtSine(size int) []Smp {
	frame := make([]Smp, size)
	for i := range size {
		frame[i] = math.Sin(2 * math.Pi * float64(i) / float64(size))
	}
	return frame
}

func wtTanh(size int) []Smp {
	frame := wtSine(size)
	for i := range frame {
		frame[i] = math.Tanh(frame[i])
	}
	return frame
}

func wtTriangle(size int) []Smp {
	frame := make([]Smp, size)
	quarter := size / 4
	for i := range quarter {
		t := float64(i) / float64(quarter)
		frame[i] = 1 - t
		frame[i+quarter] = -t
		frame[i+2*quarter] = t - 1
		frame[i+3*quarter] = t
	}
	return frame
}

func wtSquare(size int) []Smp {
	frame := make([]Smp, size)
	quarter := size / 4
	for i := range quarter {
		frame[i] = 1
		frame[i+quarter] = -1
		frame[i+2*quarter] = -1
		frame[i+3*quarter] = 1
	}
	return frame
}

func wtPulse(size int, pw float64) []Smp {
	if pw < 0 {
		pw = 0
	}
	if pw > 1 {
		pw = 1
	}
	onSamples := int(math.Round(pw * float64(size)))
	frame := make([]Smp, size)
	for i := 0; i < size; i++ {
		if i < onSamples {
			frame[i] = 1
		} else {
			frame[i] = -1
		}
	}
	return frame
}

func wtSaw(size int) []Smp {
	frame := make([]Smp, size)
	half := size / 2
	for i := range half {
		t := float64(i) / float64(half)
		frame[(i+size/4)%size] = t - 1
		frame[(i+half+size/4)%size] = t
	}
	return frame
}

func wavetableFromVal(v Val) (*Wavetable, error) {
	switch x := v.(type) {
	case *Wavetable:
		return x, nil
	case Vec:
		if len(x) == 0 {
			return nil, fmt.Errorf("wavetable: empty vector")
		}
		// If elements are vectors or tapes, treat as multi-frame. Otherwise single frame.
		switch x[0].(type) {
		case Vec, *Tape, *Wavetable:
			frames := make([][]Smp, 0, len(x))
			for i, item := range x {
				switch f := item.(type) {
				case Vec:
					wt, err := wavetableFromVec(f)
					if err != nil {
						return nil, err
					}
					frames = append(frames, wt.frames[0])
				case *Tape:
					frames = append(frames, frameFromTape(f))
				case *Wavetable:
					if len(f.frames) != 1 {
						return nil, fmt.Errorf("wavetable: nested wavetable at frame %d has %d frames; expected 1", i, len(f.frames))
					}
					frames = append(frames, f.frames[0])
				default:
					return nil, fmt.Errorf("wavetable: unsupported frame type %T at index %d", item, i)
				}
			}
			return newWavetableFromFrames(frames)
		default:
			return wavetableFromVec(x)
		}
	case *Tape:
		return newWavetableFromFrames([][]Smp{frameFromTape(x)})
	default:
		return nil, fmt.Errorf("wavetable: cannot coerce %T", v)
	}
}

func streamFromVal(v Val) (Stream, error) {
	if v == nil {
		return Num(0).Stream(), nil
	}
	if s, ok := v.(Streamable); ok {
		return s.Stream(), nil
	}
	return Stream{}, fmt.Errorf("expected streamable value, got %T", v)
}

// WavetableOsc produces a mono stream using freq and morph streams.
func WavetableOsc(freq Stream, phase float64, wt *Wavetable, morph Stream) Stream {
	return makeStream(1, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		fnext, fstop := iter.Pull(freq.Mono().seq)
		defer fstop()
		mnext, mstop := iter.Pull(morph.Mono().seq)
		defer mstop()
		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		ph := Smp(phase)
		sr := Smp(SampleRate())
		for {
			mf, mok := mnext()
			if !mok {
				return
			}
			out[0] = wt.Sample(ph, mf[0])
			if !yield(out) {
				return
			}
			f, ok := fnext()
			if !ok {
				return
			}
			inc := f[0] / sr
			ph = math.Mod(ph+inc, 1.0)
		}
	})
}

func init() {
	RegisterWord("wt", func(vm *VM) error {
		v := vm.Pop()
		wt, err := wavetableFromVal(v)
		if err != nil {
			return err
		}
		vm.Push(wt)
		return nil
	})

	RegisterWord("wt/sin", func(vm *VM) error {
		vm.Push(wtSine(DefaultWavetableSize))
		return nil
	})

	RegisterWord("wt/tanh", func(vm *VM) error {
		vm.Push(wtTanh(DefaultWavetableSize))
		return nil
	})

	RegisterWord("wt/triangle", func(vm *VM) error {
		vm.Push(wtTriangle(DefaultWavetableSize))
		return nil
	})

	RegisterWord("wt/square", func(vm *VM) error {
		vm.Push(wtSquare(DefaultWavetableSize))
		return nil
	})

	RegisterWord("wt/pulse", func(vm *VM) error {
		pw := 0.5
		if pval := vm.GetVal(":pw"); pval != nil {
			if pnum, ok := pval.(Num); ok {
				pw = float64(pnum)
			} else {
				return fmt.Errorf("wt/pulse: :pw must be number")
			}
		}
		vm.Push(wtPulse(DefaultWavetableSize, pw))
		return nil
	})

	RegisterWord("wt/saw", func(vm *VM) error {
		vm.Push(wtSaw(DefaultWavetableSize))
		return nil
	})

	RegisterWord("~wt", func(vm *VM) error {
		wtVal := vm.Pop()
		wt, err := wavetableFromVal(wtVal)
		if err != nil {
			return err
		}
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase := 0.0
		if pval := vm.GetVal(":phase"); pval != nil {
			if pnum, ok := pval.(Num); ok {
				phase = float64(pnum)
			}
		}
		morphVal := vm.GetVal(":morph")
		morphStream, err := streamFromVal(morphVal)
		if err != nil {
			// default to 0 morph
			morphStream = Num(0).Stream()
		}
		vm.Push(WavetableOsc(freq, phase, wt, morphStream))
		return nil
	})
}
