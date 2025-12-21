package main

import (
	"fmt"
	"iter"
	"math"

	"github.com/mjibson/go-dsp/fft"
)

const MaxMipLevel = 8
const BaseFrameSize = 8192

// WTFrame is a single-cycle waveform.
//
// Typically several such frames are kept at each mip level of a
// wavetable and the oscillator morphs between them as it is
// generating sound.
type WTFrame = []Smp

// Wavetable represents a collection of single-cycle frames with optional frame morphing.
// Level 0 contains the base frames; additional mip levels are built lazily on demand.
type Wavetable struct {
	mips [][]WTFrame // mips[level][frame][sample]; level 0 is the base table
}

// removeDCInPlace subtracts the mean from the frame to center it at 0.
func removeDCInPlace(frame WTFrame) {
	n := len(frame)
	if n == 0 {
		return
	}
	sum := 0.0
	for _, v := range frame {
		sum += float64(v)
	}
	mean := sum / float64(n)
	if math.Abs(mean) < 1e-12 {
		return
	}
	for i := range frame {
		frame[i] -= Smp(mean)
	}
}

func newWavetableFromFrames(baseFrames []WTFrame) (*Wavetable, error) {
	if len(baseFrames) == 0 {
		return nil, fmt.Errorf("wavetable: no frames")
	}
	baseFrameSize := len(baseFrames[0])
	if baseFrameSize == 0 {
		return nil, fmt.Errorf("wavetable: empty frame")
	}
	for i, f := range baseFrames {
		if len(f) != baseFrameSize {
			return nil, fmt.Errorf("wavetable: frame %d has size %d, expected %d", i, len(f), baseFrameSize)
		}
		removeDCInPlace(f)
	}
	wt := &Wavetable{}
	wt.mips = make([][]WTFrame, 0, MaxMipLevel)
	wt.mips = append(wt.mips, baseFrames)
	return wt, nil
}

func newWavetableFromFrame(baseFrame WTFrame) (*Wavetable, error) {
	return newWavetableFromFrames([]WTFrame{baseFrame})
}

func (wt *Wavetable) getVal() Val { return wt }

func (wt *Wavetable) String() string {
	levels := len(wt.mips)
	frames := 0
	size := 0
	if levels > 0 {
		frames = len(wt.mips[0])
		if frames > 0 {
			size = len(wt.mips[0][0])
		}
	}
	return fmt.Sprintf("Wavetable(frames=%d size=%d levels=%d)", frames, size, levels)
}

// sampleFrame returns a sample from a single frame at fractional phase [0,1).
// Uses 4-point Lagrange (Catmull-Rom) interpolation when possible; falls back to linear for very short frames.
func sampleFrame(frame WTFrame, phase Smp) Smp {
	n := len(frame)
	if n == 0 {
		return 0
	}
	p := math.Mod(float64(phase), 1.0)
	if p < 0 {
		p += 1.0
	}
	pos := p * float64(n)
	i0 := int(pos) % n
	frac := pos - float64(i0)

	// For tiny frames, just do linear.
	if n < 4 {
		i1 := (i0 + 1) % n
		return frame[i0]*(1.0-frac) + frame[i1]*frac
	}

	// 4-point Catmull-Rom (equivalent to cubic Lagrange with uniform parameterization).
	im1 := (i0 - 1 + n) % n
	i1 := (i0 + 1) % n
	i2 := (i0 + 2) % n
	t := frac
	a0 := -0.5*frame[im1] + 1.5*frame[i0] - 1.5*frame[i1] + 0.5*frame[i2]
	a1 := frame[im1] - 2.5*frame[i0] + 2.0*frame[i1] - 0.5*frame[i2]
	a2 := -0.5*frame[im1] + 0.5*frame[i1]
	a3 := frame[i0]
	return ((a0*t+a1)*t+a2)*t + a3
}

// ensureLevel builds mip level l if not present, ensuring l-1 exists first.
func (wt *Wavetable) ensureLevel(l int) {
	if l <= 0 {
		return
	}
	if l >= len(wt.mips) {
		wt.ensureLevel(l - 1)
	}
	for len(wt.mips) <= l {
		wt.mips = append(wt.mips, nil)
	}
	if wt.mips[l] != nil {
		return
	}
	prev := wt.mips[l-1]
	size := len(prev[0])
	if size <= 16 {
		wt.mips[l] = prev
		return
	}

	next := make([]WTFrame, len(prev))
	for i, frame := range prev {
		nextFrame := buildFFTLowpass(frame)
		removeDCInPlace(nextFrame)
		next[i] = nextFrame
	}
	wt.mips[l] = next
}

// buildFFTLowpass takes a frame and returns a half-size, lowpassed version using FFT bin masking.
// It zeros bins above half the Nyquist of the previous level and downsamples by 2.
func buildFFTLowpass(frame WTFrame) WTFrame {
	n := len(frame)
	if n <= 1 {
		return append(WTFrame(nil), frame...)
	}
	// FFT expects complex input.
	x := make([]complex128, n)
	for i, v := range frame {
		x[i] = complex(float64(v), 0)
	}
	X := fft.FFT(x)

	// Zero upper half of bins (simple brickwall at N/4 of original sample rate, since we will downsample by 2).
	for k := n/4 + 1; k < n-(n/4); k++ {
		X[k] = 0
	}

	// IFFT back.
	xt := fft.IFFT(X)
	// Downsample by 2 with implicit box filter from lowpass.
	nextN := n / 2
	out := make(WTFrame, nextN)
	for i := range nextN {
		// fft.IFFT divides by N; xt[2*i] has that scaling already.
		out[i] = Smp(real(xt[2*i]))
	}
	removeDCInPlace(out)
	return out
}

// selectLevel chooses a mip level based on instantaneous frequency.
// sr: sample rate, freq: Hz, baseSize: samples of level 0.
func selectLevel(freq, sr float64, baseFrameSize int) int {
	if freq <= 0 || baseFrameSize <= 0 || sr <= 0 {
		return 0
	}
	// target max harmonic that fits under Nyquist.
	H := (sr / 2.0) / freq
	if H <= 1 {
		return 0
	}
	// level = log2(baseSize / H), clamped
	return max(int(math.Log2(float64(baseFrameSize)/H)), 0)
}

// sampleFrameAtLevel samples from a specific mip level with morph.
//
// The function assumes that mip level `level` already exists.
func (wt *Wavetable) sampleFrameAtLevel(level int, phase, morph Smp) Smp {
	frames := wt.mips[level]
	if len(frames) == 0 {
		return 0
	}
	if len(frames) == 1 {
		return sampleFrame(frames[0], phase)
	}
	m := float64(morph)
	if m < 0 {
		m = 0
	}
	if m > 1 {
		m = 1
	}
	idx := m * float64(len(frames)-1)
	i0 := int(idx)
	frac := idx - float64(i0)
	i1 := i0 + 1
	if i1 >= len(frames) {
		i1 = len(frames) - 1
		frac = 0
	}
	s0 := sampleFrame(frames[i0], phase)
	s1 := sampleFrame(frames[i1], phase)
	return Smp(float64(s0)*(1.0-frac) + float64(s1)*frac)
}

// SampleMip samples using mip levels chosen from freq; crossfades between adjacent levels.
func (wt *Wavetable) SampleMip(phase, morph, freq Smp, sr float64) Smp {
	if wt == nil || len(wt.mips) == 0 || len(wt.mips[0]) == 0 || len(wt.mips[0][0]) == 0 {
		return 0
	}
	baseFrames := wt.mips[0]
	baseFrameSize := len(baseFrames[0])
	lvl := min(selectLevel(float64(freq), sr, baseFrameSize), MaxMipLevel)
	wt.ensureLevel(lvl)
	// choose second level for crossfade if available
	lvl2 := lvl
	fade := 0.0
	// simple heuristic: if fractional log places us near next level, crossfade
	// compute continuous level
	H := (sr / 2.0) / float64(freq)
	clvl := math.Log2(float64(baseFrameSize) / H)
	if clvl > float64(lvl) {
		lvl2 = lvl + 1
		fade = clvl - float64(lvl)
		if fade > 1 {
			fade = 1
		}
	}
	if lvl2 > MaxMipLevel {
		lvl2 = MaxMipLevel
	}
	wt.ensureLevel(lvl2)
	s0 := wt.sampleFrameAtLevel(lvl, phase, morph)
	if lvl2 == lvl {
		return s0
	}
	s1 := wt.sampleFrameAtLevel(lvl2, phase, morph)
	return Smp((1-fade)*float64(s0) + fade*float64(s1))
}

func wavetableFromVec(v Vec) (*Wavetable, error) {
	// Treat a flat numeric vector as one frame.
	frame := make(WTFrame, len(v))
	for i, item := range v {
		n, ok := item.(Num)
		if !ok {
			return nil, fmt.Errorf("wavetable: expected numeric vector, got %T at index %d", item, i)
		}
		frame[i] = Smp(n)
	}
	return newWavetableFromFrames([]WTFrame{frame})
}

func baseFrameFromTape(t *Tape) WTFrame {
	baseFrame := make(WTFrame, t.nframes)
	// take first channel
	for i := 0; i < t.nframes; i++ {
		baseFrame[i] = t.samples[i*t.nchannels]
	}
	return baseFrame
}

func wtSin() (*Wavetable, error) {
	size := BaseFrameSize
	baseFrame := make(WTFrame, size)
	for i := range size {
		baseFrame[i] = math.Sin(2 * math.Pi * float64(i) / float64(size))
	}
	removeDCInPlace(baseFrame)
	return newWavetableFromFrame(baseFrame)
}

func wtTanh() (*Wavetable, error) {
	wt, err := wtSin()
	if err != nil {
		return nil, err
	}
	baseFrame := wt.mips[0][0]
	for i := range baseFrame {
		baseFrame[i] = math.Tanh(baseFrame[i])
	}
	removeDCInPlace(baseFrame)
	return wt, nil
}

func wtTriangle() (*Wavetable, error) {
	size := BaseFrameSize
	baseFrame := make(WTFrame, size)
	quarter := size / 4
	for i := range quarter {
		t := float64(i) / float64(quarter)
		baseFrame[i] = 1 - t
		baseFrame[i+quarter] = -t
		baseFrame[i+2*quarter] = t - 1
		baseFrame[i+3*quarter] = t
	}
	removeDCInPlace(baseFrame)
	return newWavetableFromFrame(baseFrame)
}

func wtSquare() (*Wavetable, error) {
	size := BaseFrameSize
	baseFrame := make(WTFrame, size)
	quarter := size / 4
	for i := range quarter {
		baseFrame[i] = 1
		baseFrame[i+quarter] = -1
		baseFrame[i+2*quarter] = -1
		baseFrame[i+3*quarter] = 1
	}
	removeDCInPlace(baseFrame)
	return newWavetableFromFrame(baseFrame)
}

func wtPulse(pw float64) (*Wavetable, error) {
	if pw < 0 {
		pw = 0
	}
	if pw > 1 {
		pw = 1
	}
	size := BaseFrameSize
	onSamples := int(math.Round(pw * float64(size)))
	baseFrame := make(WTFrame, size)
	for i := range size {
		if i < onSamples {
			baseFrame[i] = 1
		} else {
			baseFrame[i] = -1
		}
	}
	removeDCInPlace(baseFrame)
	return newWavetableFromFrame(baseFrame)
}

func wtSaw() (*Wavetable, error) {
	size := BaseFrameSize
	baseFrame := make(WTFrame, size)
	half := size / 2
	for i := range half {
		t := float64(i) / float64(half)
		baseFrame[(i+size/4)%size] = t - 1
		baseFrame[(i+half+size/4)%size] = t
	}
	removeDCInPlace(baseFrame)
	return newWavetableFromFrame(baseFrame)
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
			frames := make([]WTFrame, 0, len(x))
			for i, item := range x {
				switch f := item.(type) {
				case Vec:
					wt, err := wavetableFromVec(f)
					if err != nil {
						return nil, err
					}
					firstBaseFrame := wt.mips[0][0]
					frames = append(frames, firstBaseFrame)
				case *Tape:
					frames = append(frames, baseFrameFromTape(f))
				case *Wavetable:
					if len(f.mips) == 0 || len(f.mips[0]) != 1 {
						return nil, fmt.Errorf("wavetable: nested wavetable at frame %d must have exactly 1 frame at level 0", i)
					}
					firstBaseFrame := f.mips[0][0]
					frames = append(frames, firstBaseFrame)
				default:
					return nil, fmt.Errorf("wavetable: unsupported frame type %T at index %d", item, i)
				}
			}
			return newWavetableFromFrames(frames)
		default:
			return wavetableFromVec(x)
		}
	case *Tape:
		return newWavetableFromFrame(baseFrameFromTape(x))
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

// WavetableOsc produces a mono stream using freq and morph streams, with mip selection.
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
			f, ok := fnext()
			if !ok {
				return
			}
			out[0] = wt.SampleMip(ph, mf[0], f[0], float64(sr))
			if !yield(out) {
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
		wt, err := wtSin()
		if err != nil {
			return err
		}
		vm.Push(wt)
		return nil
	})

	RegisterWord("wt/tanh", func(vm *VM) error {
		wt, err := wtTanh()
		if err != nil {
			return err
		}
		vm.Push(wt)
		return nil
	})

	RegisterWord("wt/triangle", func(vm *VM) error {
		wt, err := wtTriangle()
		if err != nil {
			return err
		}
		vm.Push(wt)
		return nil
	})

	RegisterWord("wt/square", func(vm *VM) error {
		wt, err := wtSquare()
		if err != nil {
			return err
		}
		vm.Push(wt)
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
		wt, err := wtPulse(pw)
		if err != nil {
			return err
		}
		vm.Push(wt)
		return nil
	})

	RegisterWord("wt/saw", func(vm *VM) error {
		wt, err := wtSaw()
		if err != nil {
			return err
		}
		vm.Push(wt)
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
