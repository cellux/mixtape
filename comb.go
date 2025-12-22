package main

import (
	"iter"
	"math"
)

// CombFilter applies a simple feedback comb filter to the input stream.
// delayFrames is a (potentially varying) stream specifying the delay in samples.
// feedback controls the amount of fed-back signal (-1..1 is stable).
// The output has the same channel count as the input.
func CombFilter(input Stream, delayFrames Stream, feedback float64) Stream {
	// Clamp feedback to a stable range.
	if feedback > 0.999 {
		feedback = 0.999
	} else if feedback < -0.999 {
		feedback = -0.999
	}

	nchannels := input.nchannels
	// Big enough for a couple seconds of delay.
	bufSize := max(SampleRate()*4, 1)

	return makeTransformStream([]Stream{input, delayFrames}, func(yield func(Frame) bool) {
		bufs := make([][]Smp, nchannels)
		for c := range bufs {
			bufs[c] = make([]Smp, bufSize)
		}
		writeIdx := 0

		dnext, dstop := iter.Pull(delayFrames.Mono().seq)
		defer dstop()

		out := make(Frame, nchannels)
		for frame := range input.seq {
			dframe, ok := dnext()
			if !ok {
				return
			}
			// Delay in samples (can be fractional).
			d := float64(dframe[0])
			if d < 1 {
				d = 1
			}
			if d > float64(bufSize-2) {
				d = float64(bufSize - 2)
			}

			di := int(math.Floor(d))
			frac := d - float64(di)
			for c := range nchannels {
				buf := bufs[c]
				// Read with simple linear interpolation.
				r0 := (writeIdx - di + bufSize) % bufSize
				r1 := (r0 + 1) % bufSize
				delayed := buf[r0] + Smp(frac)*(buf[r1]-buf[r0])

				y := frame[c] + Smp(feedback)*delayed
				out[c] = y
				buf[writeIdx] = y
			}

			writeIdx++
			if writeIdx == bufSize {
				writeIdx = 0
			}

			if !yield(out) {
				return
			}
		}
	})
}

func init() {
	// Stack: input_stream delay_stream feedback -> output_stream
	RegisterWord("comb", func(vm *VM) error {
		fb, err := Pop[Num](vm)
		if err != nil {
			return err
		}

		delayVal := vm.Pop()
		delayStream, err := streamFromVal(delayVal)
		if err != nil {
			return err
		}

		inputStream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}

		vm.Push(CombFilter(inputStream, delayStream, float64(fb)))
		return nil
	})
}
