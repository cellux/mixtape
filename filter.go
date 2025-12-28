package main

import "math"

// DCBlock applies a simple one-pole high-pass filter to remove DC offset.
// alpha controls the cutoff; typical small value like 0.995.
func DCBlock(s Stream, alpha float64) Stream {
	if s.nframes != 0 {
		// finite streams handled specially
		t := s.Take(nil, s.nframes)
		t.removeDCInPlace()
		return t.Stream()
	}
	return makeTransformStream([]Stream{s}, func(inputs []Stream) Stepper {
		out := make(Frame, s.nchannels)
		prevIn := make([]Smp, s.nchannels)
		prevOut := make([]Smp, s.nchannels)
		next := inputs[0].Next
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			for c := range s.nchannels {
				y := frame[c] - prevIn[c] + Smp(alpha)*prevOut[c]
				prevIn[c] = frame[c]
				prevOut[c] = y
				out[c] = y
			}
			return out, true
		}
	})
}

// OnePole applies a first-order IIR smoother: y[n] = a*y[n-1] + (1-a)*x[n]
// a is clamped to [0,1]; a=0 is passthrough, larger values increase smoothing.
func OnePole(s Stream, a float64) Stream {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	nchannels := s.nchannels
	return makeTransformStream([]Stream{s}, func(inputs []Stream) Stepper {
		prev := make(Frame, nchannels)
		out := make(Frame, nchannels)
		next := inputs[0].Next
		initialized := false
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			if !initialized {
				copy(prev, frame)
				copy(out, prev)
				initialized = true
			} else {
				for c := range nchannels {
					prev[c] = Smp(a)*prev[c] + Smp(1-a)*frame[c]
					out[c] = prev[c]
				}
			}
			return out, true
		}
	})
}

// cutoffToAlpha converts cutoff Hz to one-pole smoothing coefficient.
// Higher cutoff => smaller alpha (less smoothing).
func cutoffToAlpha(cutoff float64) float64 {
	if cutoff < 0 {
		cutoff = 0
	}
	sr := float64(SampleRate())
	if sr <= 0 {
		return 1
	}
	// a = exp(-2*pi*fc/sr)
	alpha := math.Exp(-2 * math.Pi * cutoff / sr)
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return alpha
}

// LP1 applies a first-order lowpass with cutoff in Hz.
func LP1(input, cutoff Stream) Stream {
	nchannels := input.nchannels
	return makeTransformStream([]Stream{input, cutoff}, func(inputs []Stream) Stepper {
		inNext := inputs[0].Next
		cNext := inputs[1].Mono().Next
		prev := make(Frame, nchannels)
		out := make(Frame, nchannels)
		initialized := false
		return func() (Frame, bool) {
			inFrame, ok := inNext()
			if !ok {
				return nil, false
			}
			cFrame, ok := cNext()
			if !ok {
				return nil, false
			}
			alpha := cutoffToAlpha(float64(cFrame[0]))
			if !initialized {
				copy(prev, inFrame)
				copy(out, inFrame)
				initialized = true
				return out, true
			}
			for ch := range nchannels {
				prev[ch] = Smp(alpha)*prev[ch] + Smp(1-alpha)*inFrame[ch]
				out[ch] = prev[ch]
			}
			return out, true
		}
	})
}

// HP1 applies a first-order highpass with cutoff in Hz.
func HP1(input, cutoff Stream) Stream {
	nchannels := input.nchannels
	return makeTransformStream([]Stream{input, cutoff}, func(inputs []Stream) Stepper {
		inNext := inputs[0].Next
		cNext := inputs[1].Mono().Next
		lp := make(Frame, nchannels)
		out := make(Frame, nchannels)
		initialized := false
		return func() (Frame, bool) {
			inFrame, ok := inNext()
			if !ok {
				return nil, false
			}
			cFrame, ok := cNext()
			if !ok {
				return nil, false
			}
			alpha := cutoffToAlpha(float64(cFrame[0]))
			if !initialized {
				copy(lp, inFrame)
				for ch := range nchannels {
					out[ch] = inFrame[ch] - lp[ch]
				}
				initialized = true
				return out, true
			}
			for ch := range nchannels {
				lp[ch] = Smp(alpha)*lp[ch] + Smp(1-alpha)*inFrame[ch]
				out[ch] = inFrame[ch] - lp[ch]
			}
			return out, true
		}
	})
}

func ap1Coefficient(cutoff float64) float64 {
	if cutoff < 0 {
		cutoff = 0
	}
	sr := float64(SampleRate())
	if sr <= 0 {
		return 0
	}
	ratio := cutoff / sr
	if ratio > 0.499 {
		ratio = 0.499
	}
	k := math.Tan(math.Pi * ratio)
	return (1 - k) / (1 + k)
}

// AP1 applies a first-order allpass with cutoff in Hz.
func AP1(input, cutoff Stream) Stream {
	nchannels := input.nchannels
	return makeTransformStream([]Stream{input, cutoff}, func(inputs []Stream) Stepper {
		inNext := inputs[0].Next
		cNext := inputs[1].Mono().Next
		xPrev := make(Frame, nchannels)
		yPrev := make(Frame, nchannels)
		out := make(Frame, nchannels)
		initialized := false
		return func() (Frame, bool) {
			inFrame, ok := inNext()
			if !ok {
				return nil, false
			}
			cFrame, ok := cNext()
			if !ok {
				return nil, false
			}
			coef := ap1Coefficient(float64(cFrame[0]))
			if !initialized {
				copy(xPrev, inFrame)
				copy(yPrev, inFrame)
				copy(out, inFrame)
				initialized = true
				return out, true
			}
			for ch := range nchannels {
				y := Smp(coef)*inFrame[ch] + xPrev[ch] - Smp(coef)*yPrev[ch]
				xPrev[ch] = inFrame[ch]
				yPrev[ch] = y
				out[ch] = y
			}
			return out, true
		}
	})
}

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

	return makeTransformStream([]Stream{input, delayFrames}, func(inputs []Stream) Stepper {
		bufs := make([][]Smp, nchannels)
		for c := range bufs {
			bufs[c] = make([]Smp, bufSize)
		}
		out := make(Frame, nchannels)
		inext := inputs[0].Next
		dnext := inputs[1].Mono().Next
		writeIdx := 0
		return func() (Frame, bool) {
			frame, ok := inext()
			if !ok {
				return nil, false
			}
			dframe, ok := dnext()
			if !ok {
				return nil, false
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

			return out, true
		}
	})
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

	RegisterWord("onepole", func(vm *VM) error {
		alphaNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(OnePole(stream, float64(alphaNum)))
		return nil
	})

	RegisterWord("lp1", func(vm *VM) error {
		cutoff, err := vm.GetStream(":cutoff")
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(LP1(input, cutoff))
		return nil
	})

	RegisterWord("hp1", func(vm *VM) error {
		cutoff, err := vm.GetStream(":cutoff")
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(HP1(input, cutoff))
		return nil
	})

	RegisterWord("ap1", func(vm *VM) error {
		cutoff, err := vm.GetStream(":cutoff")
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(AP1(input, cutoff))
		return nil
	})

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
