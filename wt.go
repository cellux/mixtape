package main

import (
	"fmt"
	"iter"
	"math"
)

const MaxMipLevel = 8

// Waveset is an array of waves at a given level of a wavetable
type Waveset []Wave

// Wavetable represents a collection of single-cycle waves with optional wave morphing.
// Level 0 contains the base waves; additional mip levels are built lazily on demand.
type Wavetable struct {
	mips []Waveset // mips[level][wave][sample]; level 0 is the base table
}

func newWavetableFromWaveset(baseWaves Waveset) (*Wavetable, error) {
	if len(baseWaves) == 0 {
		return nil, fmt.Errorf("wavetable: no waves")
	}
	baseWaveSize := len(baseWaves[0])
	if baseWaveSize == 0 {
		return nil, fmt.Errorf("wavetable: empty wave")
	}
	for i, w := range baseWaves {
		if len(w) != baseWaveSize {
			return nil, fmt.Errorf("wavetable: wave %d has size %d, expected %d", i, len(w), baseWaveSize)
		}
		w.removeDCInPlace()
	}
	wt := &Wavetable{}
	wt.mips = make([]Waveset, 0, MaxMipLevel)
	wt.mips = append(wt.mips, baseWaves)
	return wt, nil
}

func newWavetableFromWave(baseWave Wave) (*Wavetable, error) {
	return newWavetableFromWaveset(Waveset{baseWave})
}

func (wt *Wavetable) getVal() Val { return wt }

func (wt *Wavetable) String() string {
	levels := len(wt.mips)
	waves := 0
	size := 0
	if levels > 0 {
		waves = len(wt.mips[0])
		if waves > 0 {
			size = len(wt.mips[0][0])
		}
	}
	return fmt.Sprintf("Wavetable(waves=%d size=%d levels=%d)", waves, size, levels)
}

func (wt *Wavetable) Wave() Wave {
	if len(wt.mips) == 0 {
		return nil
	}
	baseWaveset := wt.mips[0]
	if len(baseWaveset) == 0 {
		return nil
	}
	return baseWaveset[0]
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

	next := make([]Wave, len(prev))
	for i, wave := range prev {
		nextWave := wave.buildFFTLowpass()
		nextWave.removeDCInPlace()
		next[i] = nextWave
	}
	wt.mips[l] = next
}

// selectMipLevel chooses a mip level based on instantaneous frequency.
// sr: sample rate, freq: Hz, baseSize: samples of level 0.
func selectMipLevel(freq, sr float64, baseWaveSize int) int {
	if freq <= 0 || baseWaveSize <= 0 || sr <= 0 {
		return 0
	}
	// target max harmonic that fits under Nyquist.
	H := (sr / 2.0) / freq
	if H <= 1 {
		return 0
	}
	// level = log2(baseSize / H), clamped
	return max(int(math.Log2(float64(baseWaveSize)/H)), 0)
}

// sampleWaveAtLevel samples from a specific mip level with morph.
//
// The function assumes that mip level `level` already exists.
func (wt *Wavetable) sampleWaveAtLevel(level int, phase, morph Smp) Smp {
	waves := wt.mips[level]
	if len(waves) == 0 {
		return 0
	}
	if len(waves) == 1 {
		return waves[0].sampleAt(phase)
	}
	m := float64(morph)
	if m < 0 {
		m = 0
	}
	if m > 1 {
		m = 1
	}
	idx := m * float64(len(waves)-1)
	i0 := int(idx)
	frac := idx - float64(i0)
	i1 := i0 + 1
	if i1 >= len(waves) {
		i1 = len(waves) - 1
		frac = 0
	}
	s0 := waves[i0].sampleAt(phase)
	s1 := waves[i1].sampleAt(phase)
	return Smp(float64(s0)*(1.0-frac) + float64(s1)*frac)
}

// SampleMip samples using mip levels chosen from freq; crossfades between adjacent levels.
func (wt *Wavetable) SampleMip(phase, morph, freq Smp, sr float64) Smp {
	if wt == nil || len(wt.mips) == 0 || len(wt.mips[0]) == 0 || len(wt.mips[0][0]) == 0 {
		return 0
	}
	baseWaves := wt.mips[0]
	baseWaveSize := len(baseWaves[0])
	lvl := min(selectMipLevel(float64(freq), sr, baseWaveSize), MaxMipLevel)
	wt.ensureLevel(lvl)
	// choose second level for crossfade if available
	lvl2 := lvl
	fade := 0.0
	// simple heuristic: if fractional log places us near next level, crossfade
	// compute continuous level
	H := (sr / 2.0) / float64(freq)
	clvl := math.Log2(float64(baseWaveSize) / H)
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
	s0 := wt.sampleWaveAtLevel(lvl, phase, morph)
	if lvl2 == lvl {
		return s0
	}
	s1 := wt.sampleWaveAtLevel(lvl2, phase, morph)
	return Smp((1-fade)*float64(s0) + fade*float64(s1))
}

func wtSin() (*Wavetable, error) {
	return newWavetableFromWave(sinWave(0))
}

func wtTanh() (*Wavetable, error) {
	return newWavetableFromWave(tanhWave(0))
}

func wtTriangle() (*Wavetable, error) {
	return newWavetableFromWave(triangleWave(0))
}

func wtSquare() (*Wavetable, error) {
	return newWavetableFromWave(squareWave(0))
}

func wtPulse(pw float64) (*Wavetable, error) {
	return newWavetableFromWave(pulseWave(0, pw))
}

func wtSaw() (*Wavetable, error) {
	return newWavetableFromWave(sawWave(0))
}

func wavetableFromVal(v Val) (*Wavetable, error) {
	switch x := v.(type) {
	case *Wavetable:
		return x, nil
	case Vec:
		if len(x) == 0 {
			return nil, fmt.Errorf("wavetable: empty vector")
		}
		// If elements can provide waves, build wavetable from waveset, otherwise single wave.
		switch x[0].(type) {
		case WaveProvider:
			waves := make(Waveset, 0, len(x))
			for i, item := range x {
				if wp, ok := x[i].(WaveProvider); ok {
					waves = append(waves, wp.Wave())
				} else {
					return nil, fmt.Errorf("wavetable: unsupported wave type %T at index %d", item, i)
				}
			}
			return newWavetableFromWaveset(waves)
		case Num:
			return newWavetableFromWave(x.Wave())
		default:
			return nil, fmt.Errorf("wavetable: cannot create wave from Vec of %T", x[0])
		}
	case WaveProvider:
		return newWavetableFromWave(x.Wave())
	case Streamable:
		s := x.Stream()
		if s.nframes == 0 {
			return nil, fmt.Errorf("wavetable: input is non-finite stream")
		}
		return wavetableFromVal(s.Take(s.nframes))
	default:
		return nil, fmt.Errorf("wavetable: cannot create wavetable from %T", v)
	}
}

// WavetableOsc produces a mono stream using freq and morph streams, with mip selection.
func WavetableOsc(freq Stream, phase float64, wt *Wavetable, morph Stream) Stream {
	return makeStream(1, 0, func(yield func(Frame) bool) {
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

// FMOsc implements phase modulation (FM) using a wavetable.
// The mod stream is in cycles, not Hz. Index is a multiplier on the mod signal.
func FMOsc(wt *Wavetable, freq Stream, mod Stream, index Stream, phase float64) Stream {
	return makeStream(1, 0, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		fnext, fstop := iter.Pull(freq.Mono().seq)
		defer fstop()
		mnext, mstop := iter.Pull(mod.Mono().seq)
		defer mstop()
		inext, istop := iter.Pull(index.Mono().seq)
		defer istop()

		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		ph := Smp(phase)
		sr := Smp(SampleRate())

		for {
			mframe, mok := mnext()
			if !mok {
				return
			}
			iframe, iok := inext()
			if !iok {
				return
			}
			fframe, fok := fnext()
			if !fok {
				return
			}

			pmPhase := ph + iframe[0]*mframe[0]
			out[0] = wt.SampleMip(pmPhase, 0, fframe[0], float64(sr))

			if !yield(out) {
				return
			}

			inc := fframe[0] / sr
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

	RegisterWord("~fm", func(vm *VM) error {
		wtVal := vm.Pop()
		wt, err := wavetableFromVal(wtVal)
		if err != nil {
			return err
		}

		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}

		mod, err := vm.GetStream(":mod")
		if err != nil {
			return err
		}

		index := Num(1).Stream()
		if v := vm.GetVal(":index"); v != nil {
			idxStream, err := streamFromVal(v)
			if err != nil {
				return err
			}
			index = idxStream
		}

		phase := 0.0
		if pval := vm.GetVal(":phase"); pval != nil {
			if pnum, ok := pval.(Num); ok {
				phase = float64(pnum)
			} else {
				return fmt.Errorf("fm: :phase must be number")
			}
		}

		vm.Push(FMOsc(wt, freq, mod, index, phase))
		return nil
	})
}
