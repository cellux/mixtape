package main

import (
	"iter"
	"math"
)

// equalPowerPan returns gains for left/right given pan in [-1,1].
func equalPowerPan(p float64) (float64, float64) {
	if p < -1 {
		p = -1
	}
	if p > 1 {
		p = 1
	}
	// map p=-1..1 -> theta in [0..pi/2]
	theta := (p + 1) * math.Pi / 4
	return math.Cos(theta), math.Sin(theta)
}

// Pan applies equal-power panning to a mono stream, returning stereo.
// Pan value can be a Num or Streamable providing values in [-1..1].
func Pan(s Stream, pan Stream) Stream {
	return makeStream(2, s.nframes, func(yield func(Frame) bool) {
		out := make(Frame, 2)
		snext, sstop := iter.Pull(s.Mono().seq)
		pnext, pstop := iter.Pull(pan.Mono().seq)
		defer sstop()
		defer pstop()
		for {
			sframe, ok := snext()
			if !ok {
				return
			}
			pframe, ok := pnext()
			if !ok {
				return
			}
			l, r := equalPowerPan(float64(pframe[0]))
			out[0] = sframe[0] * Smp(l)
			out[1] = sframe[0] * Smp(r)
			if !yield(out) {
				return
			}
		}
	})
}

func init() {
	RegisterWord("pan", func(vm *VM) error {
		// input pan -- output
		pan, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(Pan(input, pan))
		return nil
	})
}
