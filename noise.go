package main

import (
	"fmt"
	"math"
	"math/bits"
)

// noiseStream returns a mono infinite stream of deterministic white noise in [-1,1].
// Uses xorshift32 PRNG with provided seed (seed 0 is mapped to state 1 to avoid lockup).
func noiseStream(seed int) Stream {
	state := uint32(seed)
	if state == 0 {
		state = 1
	}
	out := make(Frame, 1)
	return makeStream(1, 0, func() (Frame, bool) {
		// xorshift32
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		u := float64(state) / float64(^uint32(0))
		out[0] = Smp(2*u - 1)
		return out, true
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
	out := make(Frame, 1)
	return makeStream(1, 0, func() (Frame, bool) {
		counter++
		tz := bits.TrailingZeros32(counter)
		if tz >= nrows {
			tz = nrows - 1
		}

		newVal := nextWhite()
		sum += newVal - rows[tz]
		rows[tz] = newVal

		out[0] = sum / Smp(nrows)
		return out, true
	})
}

// brownStream returns a mono infinite stream of deterministic brown noise in [-1,1].
// Each sample is a clamped random walk step of size `step` on uniform noise.
func brownStream(seed int, step Smp) Stream {
	state := uint32(seed)
	if state == 0 {
		state = 1
	}

	x := Smp(0)
	out := make(Frame, 1)
	return makeStream(1, 0, func() (Frame, bool) {
		// xorshift32
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		u := float64(state) / float64(^uint32(0))

		x += step * Smp(2*u-1)
		x = math.Min(1, math.Max(-1, x))

		out[0] = x
		return out, true
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

	RegisterWord("~brown", func(vm *VM) error {
		stepNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}

		seed := 0
		if sval := vm.GetVal(":seed"); sval != nil {
			if snum, ok := sval.(Num); ok {
				seed = int(snum)
			} else {
				return fmt.Errorf("brown: :seed must be number")
			}
		}

		vm.Push(brownStream(seed, Smp(stepNum)))
		return nil
	})
}
