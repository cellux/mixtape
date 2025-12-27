package main

import (
	"fmt"
)

// Coefficients from Vital's IIR halfband decimator (src/synthesis/filters/iir_halfband_decimator.cpp).
var (
	decimTaps9A  = []float64{0.167135116548925, 0.742130012538075}
	decimTaps9B  = []float64{0.0413554705262319, 0.3878932830211427}
	decimTaps25A = []float64{0.093022421467960, 0.312318050871736, 0.548379093159427, 0.737198546150414, 0.872234992057129, 0.975497791832324}
	decimTaps25B = []float64{0.024388383731296, 0.194029987625265, 0.433855675727187, 0.650124972769370, 0.810418671775866, 0.925979700943193}
)

// halfbandStage holds per-channel state for a single 2x IIR halfband decimation stage.
type halfbandStage struct {
	tapsA []float64
	tapsB []float64
	inA   [][]Smp
	outA  [][]Smp
	inB   [][]Smp
	outB  [][]Smp
}

func newHalfbandStage(nchannels int, tapsA, tapsB []float64) *halfbandStage {
	s := &halfbandStage{
		tapsA: tapsA,
		tapsB: tapsB,
		inA:   make([][]Smp, nchannels),
		outA:  make([][]Smp, nchannels),
		inB:   make([][]Smp, nchannels),
		outB:  make([][]Smp, nchannels),
	}
	for c := range nchannels {
		s.inA[c] = make([]Smp, len(tapsA))
		s.outA[c] = make([]Smp, len(tapsA))
		s.inB[c] = make([]Smp, len(tapsB))
		s.outB[c] = make([]Smp, len(tapsB))
	}
	return s
}

// step consumes two input samples for channel c and returns one decimated sample.
func (s *halfbandStage) step(c int, x0, x1 Smp) Smp {
	r0 := x0
	r1 := x1
	for i, coef := range s.tapsA {
		delta := r0 - s.outA[c][i]
		newR := s.inA[c][i]*Smp(coef) + delta
		s.inA[c][i] = r0
		s.outA[c][i] = newR
		r0 = newR
	}
	for i, coef := range s.tapsB {
		delta := r1 - s.outB[c][i]
		newR := s.inB[c][i]*Smp(coef) + delta
		s.inB[c][i] = r1
		s.outB[c][i] = newR
		r1 = newR
	}
	return 0.5 * (r0 + r1)
}

// iirHalfbandStage builds a stream that decimates by 2 using the provided taps.
func iirHalfbandStage(input Stream, tapsA, tapsB []float64) Stream {
	nchannels := input.nchannels
	nframes := 0
	if input.nframes > 0 {
		nframes = input.nframes / 2
	}

	return makeRewindableStream(nchannels, nframes, func() Stepper {
		stage := newHalfbandStage(nchannels, tapsA, tapsB)
		next := input.clone().Next
		out := make(Frame, nchannels)
		return func() (Frame, bool) {
			f0, ok := next()
			if !ok {
				return nil, false
			}
			f1, ok := next()
			if !ok {
				return nil, false
			}
			for c := range nchannels {
				out[c] = stage.step(c, f0[c], f1[c])
			}
			return out, true
		}
	})
}

// Decimate applies Vital's cascaded halfband IIR decimation by the given power-of-two factor.
// If sharp is true, the final stage uses the longer (25-tap) coefficients; earlier stages use 9-tap.
func Decimate(input Stream, factor int, sharp bool) Stream {
	if factor <= 1 {
		return input
	}

	// Expect power-of-two factor; caller should validate.
	stages := 0
	for f := factor; f > 1; f >>= 1 {
		stages++
	}

	out := input.clone()
	for i := 0; i < stages; i++ {
		last := i == stages-1
		useSharp := sharp && last
		tapsA := decimTaps9A
		tapsB := decimTaps9B
		if useSharp {
			tapsA = decimTaps25A
			tapsB = decimTaps25B
		}
		out = iirHalfbandStage(out, tapsA, tapsB)
	}
	return out
}

func init() {
	// Stack: input_stream factor sharp_flag -> output_stream
	// factor must be a power of two; sharp_flag != 0 enables the sharper final stage.
	RegisterWord("vital/decimate", func(vm *VM) error {
		sharpNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		factorNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		factor := int(factorNum)
		if factor < 1 {
			return fmt.Errorf("decim: factor must be >= 1, got %d", factor)
		}
		if factor&(factor-1) != 0 {
			return fmt.Errorf("decim: factor must be a power of two, got %d", factor)
		}
		sharp := sharpNum != 0
		vm.Push(Decimate(input, factor, sharp))
		return nil
	})
}
