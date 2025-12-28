package main

import "math"

// DCMeanCenter subtracts the per-channel mean for finite streams.
func DCMeanCenter(s Stream) Stream {
	t := s.Take(nil, s.nframes)
	t.removeDCInPlace()
	return t.Stream()
}

// DCBlock applies a simple one-pole high-pass filter to remove DC offset.
// alpha controls the cutoff; typical small value like 0.995.
func DCBlock(s Stream, alpha float64) Stream {
	if s.nframes != 0 {
		return DCMeanCenter(s)
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

// DCFilter implements Vital's dc_filter: y[n] = (x[n]-x[n-1]) + a*y[n-1]
// with a = 1 - (1 / sampleRate). It's a very low cutoff (~0.16 Hz @ 48kHz).
func DCFilter(s Stream) Stream {
	alpha := 1.0 - 1.0/float64(SampleRate())
	return DCBlock(s, alpha)
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

func ap2Coefficients(cutoffHz, q float64) (b0, b1, b2, a1, a2 float64) {
	sr := float64(SampleRate())
	if sr <= 0 {
		// Identity.
		return 1, 0, 0, 0, 0
	}
	if cutoffHz < 0 {
		cutoffHz = 0
	}
	if q < 1e-6 {
		q = 1e-6
	}

	ratio := cutoffHz / sr
	// Avoid degenerate sin(0)=0 / sin(pi)=0 cases.
	if ratio < 1e-6 {
		ratio = 1e-6
	}
	if ratio > 0.499 {
		ratio = 0.499
	}

	w0 := 2 * math.Pi * ratio
	cosw0 := math.Cos(w0)
	sinw0 := math.Sin(w0)
	alpha := sinw0 / (2 * q)

	// RBJ cookbook allpass biquad.
	bb0 := 1 - alpha
	bb1 := -2 * cosw0
	bb2 := 1 + alpha
	aa0 := 1 + alpha
	aa1 := -2 * cosw0
	aa2 := 1 - alpha

	b0 = bb0 / aa0
	b1 = bb1 / aa0
	b2 = bb2 / aa0
	a1 = aa1 / aa0
	a2 = aa2 / aa0
	return
}

// AP2 applies a second-order allpass (biquad) with cutoff in Hz and Q.
func AP2(input, cutoff, q Stream) Stream {
	nchannels := input.nchannels
	return makeTransformStream([]Stream{input, cutoff, q}, func(inputs []Stream) Stepper {
		inNext := inputs[0].Next
		cNext := inputs[1].Mono().Next
		qNext := inputs[2].Mono().Next

		x1 := make([]Smp, nchannels)
		x2 := make([]Smp, nchannels)
		y1 := make([]Smp, nchannels)
		y2 := make([]Smp, nchannels)
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
			qFrame, ok := qNext()
			if !ok {
				return nil, false
			}

			if !initialized {
				// Initialize history so constant signals pass through unchanged.
				// (Also matches AP1's first-sample passthrough behavior.)
				for ch := range nchannels {
					x1[ch] = inFrame[ch]
					x2[ch] = inFrame[ch]
					y1[ch] = inFrame[ch]
					y2[ch] = inFrame[ch]
					out[ch] = inFrame[ch]
				}
				initialized = true
				return out, true
			}

			b0, b1, b2, a1, a2 := ap2Coefficients(float64(cFrame[0]), float64(qFrame[0]))
			for ch := range nchannels {
				x := inFrame[ch]
				y := Smp(b0)*x + Smp(b1)*x1[ch] + Smp(b2)*x2[ch] - Smp(a1)*y1[ch] - Smp(a2)*y2[ch]
				x2[ch] = x1[ch]
				x1[ch] = x
				y2[ch] = y1[ch]
				y1[ch] = y
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

// digitalSVFState holds per-channel integrator state for the SVF.
type digitalSVFState struct {
	ic1eq []Smp
	ic2eq []Smp
}

func newDigitalSVFState(nchannels int) *digitalSVFState {
	return &digitalSVFState{
		ic1eq: make([]Smp, nchannels),
		ic2eq: make([]Smp, nchannels),
	}
}

// svfMix maps a blend in [-1,1] to low/band/high amounts
// blend < 0 favours lowpass, > 0 favours highpass, 0 gives bandpass.
func svfMix(blend Smp) (low, band, high Smp) {
	if blend < -1 {
		blend = -1
	} else if blend > 1 {
		blend = 1
	}
	// Band amount follows a circular crossfade to keep unity energy.
	band = math.Sqrt(math.Max(0, 1-math.Pow(blend, 2)))
	if blend < 0 {
		low = -blend
		high = 0
	} else {
		low = 0
		high = blend
	}
	return
}

// svfCoefficient computes the one-pole SVF coefficient: tan(pi * min(0.499, f/sr)).
func svfCoefficient(cutoffHz Smp) Smp {
	sr := float64(SampleRate())
	ratio := float64(cutoffHz) / sr
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 0.499 {
		ratio = 0.499
	}
	return Smp(math.Tan(math.Pi * ratio))
}

// svfStepper returns a stepper that yields the LP/BP/HP outputs of the TPT SVF.
func svfStepper(input, cutoff, resonance, drive Stream) func() (lpf, bpf, hpf Frame, valid bool) {
	nchannels := input.nchannels

	inNext := input.Next
	cNext := cutoff.Next
	rNext := resonance.Next
	dNext := drive.Next

	state := newDigitalSVFState(nchannels)
	lp := make(Frame, nchannels)
	bp := make(Frame, nchannels)
	hp := make(Frame, nchannels)

	return func() (Frame, Frame, Frame, bool) {
		inFrame, ok := inNext()
		if !ok {
			return nil, nil, nil, false
		}
		cFrame, ok := cNext()
		if !ok {
			return nil, nil, nil, false
		}
		rFrame, ok := rNext()
		if !ok {
			return nil, nil, nil, false
		}
		dFrame, ok := dNext()
		if !ok {
			return nil, nil, nil, false
		}

		cut := cFrame[0]
		res := rFrame[0]
		drv := dFrame[0]

		// Clamp resonance to avoid division by zero.
		if res < 1e-6 {
			res = 1e-6
		}
		k := Smp(1) / res
		g := svfCoefficient(cut)

		// TPT SVF coefficients
		denom := Smp(1) + g*(g+k)
		if denom == 0 {
			denom = 1e-9
		}
		a0 := Smp(1) / denom // a1 in Simper paper
		a1 := g * a0         // a2
		a2 := g * a1         // a3

		for c := range nchannels {
			x := drv * inFrame[c]

			// Topology-preserving transform (TPT) SVF (Simper):
			//
			// a1 = 1/(1 + g*(g+k))
			// a2 = g*a1
			// a3 = g*a2
			// v3 = x - ic2eq
			// v1 = a1*ic1eq + a2*v3
			// v2 = ic2eq + a2*ic1eq + a3*v3
			// ic1eq = 2*v1 - ic1eq
			// ic2eq = 2*v2 - ic2eq
			v3 := x - state.ic2eq[c]
			v1 := a0*state.ic1eq[c] + a1*v3
			v2 := state.ic2eq[c] + a1*state.ic1eq[c] + a2*v3
			state.ic1eq[c] = 2*v1 - state.ic1eq[c]
			state.ic2eq[c] = 2*v2 - state.ic2eq[c]

			lp[c] = v2
			bp[c] = v1
			hp[c] = x - k*bp[c] - lp[c]
		}

		return lp, bp, hp, true
	}
}

// DigitalSVF applies a Vital-inspired digital state-variable filter.
// Parameters are streams to allow modulation:
//
//	input:     audio input (N channels)
//	cutoff:    cutoff frequency in Hz (mono stream)
//	resonance: resonance (Q). Values <= 0 are clamped to a small epsilon.
//	drive:     input drive multiplier.
//	blend:     blend in [-1,1], mapping lowpass(-1) -> bandpass(0) -> highpass(+1).
func DigitalSVF(input, cutoff, resonance, drive, blend Stream) Stream {
	nchannels := input.nchannels

	// Let makeTransformStream compute nframes as the shortest among inputs.
	return makeTransformStream([]Stream{input, cutoff, resonance, drive, blend}, func(inputs []Stream) Stepper {
		sInput := inputs[0]
		sCutoff := inputs[1].Mono()
		sResonance := inputs[2].Mono()
		sDrive := inputs[3].Mono()
		step := svfStepper(sInput, sCutoff, sResonance, sDrive)

		sBlend := inputs[4].Mono()
		bNext := sBlend.Next

		out := make(Frame, nchannels)

		return func() (Frame, bool) {
			lp, bp, hp, ok := step()
			if !ok {
				return nil, false
			}
			bFrame, ok := bNext()
			if !ok {
				return nil, false
			}

			blendVal := bFrame[0]
			lowAmt, bandAmt, highAmt := svfMix(blendVal)

			for c := range nchannels {
				y := lowAmt*lp[c] + bandAmt*bp[c] + highAmt*hp[c]
				out[c] = y
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

	RegisterWord("dc", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(DCFilter(stream))
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

	RegisterWord("ap2", func(vm *VM) error {
		cutoff, err := vm.GetStream(":cutoff")
		if err != nil {
			return err
		}
		q, err := vm.GetStream(":q")
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(AP2(input, cutoff, q))
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

	RegisterWord("svf", func(vm *VM) error {
		blend, err := vm.GetStream(":blend")
		if err != nil {
			return err
		}
		drive, err := vm.GetStream(":drive")
		if err != nil {
			return err
		}
		resonance, err := vm.GetStream(":q")
		if err != nil {
			return err
		}
		cutoff, err := vm.GetStream(":cutoff")
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(DigitalSVF(input, cutoff, resonance, drive, blend))
		return nil
	})
}
