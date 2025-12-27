package main

import (
	"fmt"
	"math"
)

const MaxMipLevel = 8

// Waveset is an array of single-channel tapes at a given level of a wavetable
type Waveset []*Tape

// Wavetable represents a collection of single-cycle waves with optional wave morphing.
// Level 0 contains the base waves; additional mip levels are built lazily on demand.
type Wavetable struct {
	mips []Waveset // mips[level][wave][sample]; level 0 is the base table
}

func newWavetableFromWaveset(baseWaves Waveset) (*Wavetable, error) {
	if len(baseWaves) == 0 {
		return nil, fmt.Errorf("wavetable: no waves")
	}
	baseWaveSize := baseWaves[0].nframes
	if baseWaveSize == 0 {
		return nil, fmt.Errorf("wavetable: empty wave")
	}
	for i, t := range baseWaves {
		if t.nframes != baseWaveSize {
			return nil, fmt.Errorf("wavetable: wave %d has size %d, expected %d", i, t.nframes, baseWaveSize)
		}
		t.removeDCInPlace()
	}
	wt := &Wavetable{}
	wt.mips = make([]Waveset, 0, MaxMipLevel)
	wt.mips = append(wt.mips, baseWaves)
	return wt, nil
}

func newWavetableFromWave(baseWave *Tape) (*Wavetable, error) {
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
			size = wt.mips[0][0].nframes
		}
	}
	return fmt.Sprintf("Wavetable(waves=%d size=%d levels=%d)", waves, size, levels)
}

func (wt *Wavetable) Tape() *Tape {
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
	size := prev[0].nframes
	if size <= 16 {
		wt.mips[l] = prev
		return
	}

	next := make(Waveset, len(prev))
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
		out := Frame{0}
		waves[0].GetInterpolatedFrameAtPhase(float64(phase), out)
		return out[0]
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
	f0 := Frame{0}
	f1 := Frame{0}
	waves[i0].GetInterpolatedFrameAtPhase(float64(phase), f0)
	waves[i1].GetInterpolatedFrameAtPhase(float64(phase), f1)
	s0 := f0[0]
	s1 := f1[0]
	return s0*(1.0-frac) + s1*frac
}

// SampleMip samples using mip levels chosen from freq; crossfades between adjacent levels.
func (wt *Wavetable) SampleMip(phase, morph, freq float64, sr float64) Smp {
	if wt == nil || len(wt.mips) == 0 || len(wt.mips[0]) == 0 || wt.mips[0][0].nframes == 0 {
		return 0
	}
	baseWaves := wt.mips[0]
	baseWaveSize := baseWaves[0].nframes
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
	return (1-fade)*s0 + fade*s1
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
		case TapeProvider:
			waves := make(Waveset, 0, len(x))
			for i, item := range x {
				if wp, ok := x[i].(TapeProvider); ok {
					waves = append(waves, wp.Tape())
				} else {
					return nil, fmt.Errorf("wavetable: unsupported wave type %T at index %d", item, i)
				}
			}
			return newWavetableFromWaveset(waves)
		case Num:
			return newWavetableFromWave(x.Tape())
		default:
			return nil, fmt.Errorf("wavetable: cannot create wave from Vec of %T", x[0])
		}
	case TapeProvider:
		return newWavetableFromWave(x.Tape())
	case Streamable:
		s := x.Stream()
		if s.nframes == 0 {
			return nil, fmt.Errorf("wavetable: input is non-finite stream")
		}
		return wavetableFromVal(s.Take(nil, s.nframes))
	default:
		return nil, fmt.Errorf("wavetable: cannot create wavetable from %T", v)
	}
}

// WavetableOsc produces a mono stream using freq and morph streams, with mip selection.
func WavetableOsc(freq Stream, phase float64, wt *Wavetable, morph Stream) Stream {
	fnext := freq.Mono().Next
	mnext := morph.Mono().Next
	if phase < 0.0 || phase >= 1.0 {
		phase = 0.0
	}
	ph := Smp(phase)
	sr := Smp(SampleRate())
	out := make(Frame, 1)
	return makeStream(1, 0, func() (Frame, bool) {
		mframe, mok := mnext()
		if !mok {
			return nil, false
		}
		fframe, fok := fnext()
		if !fok {
			return nil, false
		}
		out[0] = wt.SampleMip(ph, mframe[0], fframe[0], float64(sr))
		inc := fframe[0] / sr
		ph = math.Mod(ph+inc, 1.0)
		return out, true
	})
}

// FMOsc implements phase modulation (FM) using a wavetable.
// The mod stream is in cycles, not Hz. Index is a multiplier on the mod signal.
func FMOsc(wt *Wavetable, freq Stream, mod Stream, index Stream, phase float64) Stream {
	fnext := freq.Mono().Next
	mnext := mod.Mono().Next
	inext := index.Mono().Next
	if phase < 0.0 || phase >= 1.0 {
		phase = 0.0
	}
	ph := Smp(phase)
	sr := Smp(SampleRate())
	out := make(Frame, 1)
	return makeStream(1, 0, func() (Frame, bool) {
		mframe, mok := mnext()
		if !mok {
			return nil, false
		}
		iframe, iok := inext()
		if !iok {
			return nil, false
		}
		fframe, fok := fnext()
		if !fok {
			return nil, false
		}

		pmPhase := ph + iframe[0]*mframe[0]
		out[0] = wt.SampleMip(pmPhase, 0, fframe[0], float64(sr))

		inc := fframe[0] / sr
		ph = math.Mod(ph+inc, 1.0)
		return out, true
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
