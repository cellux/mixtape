package main

// DCBlock applies a simple one-pole high-pass filter to remove DC offset.
// alpha controls the cutoff; typical small value like 0.995.
func DCBlock(s Stream, alpha float64) Stream {
	return makeTransformStream([]Stream{s}, func(yield func(Frame) bool) {
		out := make(Frame, s.nchannels)
		prevIn := make([]Smp, s.nchannels)
		prevOut := make([]Smp, s.nchannels)
		for frame := range s.seq {
			for c := range s.nchannels {
				y := frame[c] - prevIn[c] + Smp(alpha)*prevOut[c]
				prevIn[c] = frame[c]
				prevOut[c] = y
				out[c] = y
			}
			if !yield(out) {
				return
			}
		}
	})
}

// DCFilter implements Vital's dc_filter: y[n] = (x[n]-x[n-1]) + a*y[n-1]
// with a = 1 - (1 / sampleRate). It's a very low cutoff (~0.16 Hz @ 48kHz).
func DCFilter(s Stream) Stream {
	alpha := 1.0 - 1.0/float64(SampleRate())
	return DCBlock(s, alpha)
}

func init() {
	RegisterWord("dc*", func(vm *VM) error {
		alphaNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		alpha := float64(alphaNum)
		vm.Push(DCBlock(stream, alpha))
		return nil
	})

	RegisterWord("dc", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(DCFilter(stream))
		return nil
	})
}
