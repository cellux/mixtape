package main

import (
	"fmt"
	"math/bits"
)

// noiseStream returns a mono infinite stream of deterministic white noise in [-1,1].
// Uses xorshift32 PRNG with provided seed (seed 0 is mapped to state 1 to avoid lockup).
func noiseStream(seed int) Stream {
	state := uint32(seed)
	if state == 0 {
		state = 1
	}
	return makeStream(1, 0, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		for {
			// xorshift32
			state ^= state << 13
			state ^= state >> 17
			state ^= state << 5
			u := float64(state) / float64(^uint32(0))
			out[0] = Smp(2*u - 1)
			if !yield(out) {
				return
			}
		}
	})
}

// pinkStream returns a mono infinite stream of deterministic pink noise using the
// Voss-McCartney algorithm with 16 rows.
func pinkStream(seed int) Stream {
	const nrows = 16
	state := uint32(seed)
	if state == 0 {
		state = 1
	}

	nextWhite := func() Smp {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		u := float64(state) / float64(^uint32(0))
		return Smp(2*u - 1)
	}

	rows := [nrows]Smp{}
	sum := Smp(0)
	for i := range nrows {
		rows[i] = nextWhite()
		sum += rows[i]
	}

	counter := uint32(0)
	return makeStream(1, 0, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		for {
			counter++
			tz := bits.TrailingZeros32(counter)
			if tz >= nrows {
				tz = nrows - 1
			}

			newVal := nextWhite()
			sum += newVal - rows[tz]
			rows[tz] = newVal

			out[0] = sum / Smp(nrows)
			if !yield(out) {
				return
			}
		}
	})
}

func init() {
	RegisterWord("~noise", func(vm *VM) error {
		seed := 0
		if sval := vm.GetVal(":seed"); sval != nil {
			if snum, ok := sval.(Num); ok {
				seed = int(snum)
			} else {
				return fmt.Errorf("noise: :seed must be number")
			}
		}
		vm.Push(noiseStream(seed))
		return nil
	})

	RegisterWord("~pink", func(vm *VM) error {
		seed := 0
		if sval := vm.GetVal(":seed"); sval != nil {
			if snum, ok := sval.(Num); ok {
				seed = int(snum)
			} else {
				return fmt.Errorf("pink: :seed must be number")
			}
		}
		vm.Push(pinkStream(seed))
		return nil
	})
}
