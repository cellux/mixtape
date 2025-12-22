package main

import "fmt"

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
}
