package main

import "math"

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
// It also returns k = 1/Q (where Q is the resonance stream), which is useful for
// derived responses like notch/peak. A caller may pass an existing state (for example
// to prime the integrators); if nil, a fresh state is allocated.
func svfStepper(input, cutoff, resonance Stream, state *digitalSVFState) func() (lpf, bpf, hpf Frame, k Smp, valid bool) {
	nchannels := input.nchannels

	inNext := input.Next
	cNext := cutoff.Next
	rNext := resonance.Next

	if state == nil {
		state = newDigitalSVFState(nchannels)
	}
	lp := make(Frame, nchannels)
	bp := make(Frame, nchannels)
	hp := make(Frame, nchannels)

	return func() (Frame, Frame, Frame, Smp, bool) {
		inFrame, ok := inNext()
		if !ok {
			return nil, nil, nil, 0, false
		}
		cFrame, ok := cNext()
		if !ok {
			return nil, nil, nil, 0, false
		}
		rFrame, ok := rNext()
		if !ok {
			return nil, nil, nil, 0, false
		}

		cut := cFrame[0]
		res := rFrame[0]

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
			x := inFrame[c]

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

		return lp, bp, hp, k, true
	}
}

// AP2 applies a second-order allpass (SVF-derived) with cutoff in Hz and Q.
// Implemented from the same TPT SVF core used by lp2/bp2/hp2/notch2/peak2.
func AP2(input, cutoff, q Stream) Stream {
	nchannels := input.nchannels

	return makeTransformStream([]Stream{input, cutoff, q}, func(inputs []Stream) Stepper {
		sInput := inputs[0]
		sCutoff := inputs[1].Mono()
		sResonance := inputs[2].Mono()

		// Use a shared SVF state so we can peek at the first frame and then continue
		// seamlessly without resetting integrators.
		state := newDigitalSVFState(nchannels)
		step := svfStepper(sInput, sCutoff, sResonance, state)
		out := make(Frame, nchannels)
		first := true

		return func() (Frame, bool) {
			lp, bp, hp, k, ok := step()
			if !ok {
				return nil, false
			}

			if first {
				// Preserve AP2's original first-sample passthrough behavior and
				// seed the SVF state so constant inputs stay constant immediately.
				for c := range nchannels {
					// Recover input: x = hp + k*bp + lp (from SVF definitions).
					x := hp[c] + k*bp[c] + lp[c]
					out[c] = x
					state.ic1eq[c] = 0
					state.ic2eq[c] = x
				}
				first = false
				return out, true
			}

			for c := range nchannels {
				// Allpass transfer: hp - k*bp + lp -> numerator (s^2 - k*s + 1)/(s^2 + k*s + 1)
				out[c] = hp[c] - k*bp[c] + lp[c]
			}
			return out, true
		}
	})
}

// Notch2 applies a 2-pole notch derived from the digital SVF core.
// Parameters are streams to allow modulation:
//
//	input:     audio input (N channels)
//	cutoff:    cutoff frequency in Hz (mono stream)
//	resonance: resonance (Q). Values <= 0 are clamped to a small epsilon.
func Notch2(input, cutoff, resonance Stream) Stream {
	nchannels := input.nchannels

	return makeTransformStream([]Stream{input, cutoff, resonance}, func(inputs []Stream) Stepper {
		sInput := inputs[0]
		sCutoff := inputs[1].Mono()
		sResonance := inputs[2].Mono()
		step := svfStepper(sInput, sCutoff, sResonance, nil)

		out := make(Frame, nchannels)

		return func() (Frame, bool) {
			lp, _, hp, _, ok := step()
			if !ok {
				return nil, false
			}

			for c := range nchannels {
				out[c] = lp[c] + hp[c]
			}

			return out, true
		}
	})
}

// Peak2 applies a peaking (bell) EQ derived from the digital SVF core.
//
// This is implemented as: y = x + (A - 1) * (k * bp)
// where A is a linear gain multiplier (typically db(:gain)), and k = 1/Q.
//
// Parameters are streams to allow modulation:
//
//	input:     audio input (N channels)
//	cutoff:    cutoff/center frequency in Hz (mono stream)
//	resonance: resonance (Q). Values <= 0 are clamped to a small epsilon.
//	gain:      linear gain multiplier (mono stream). A=1 is neutral; >1 boosts; <1 cuts.
func Peak2(input, cutoff, resonance, gain Stream) Stream {
	nchannels := input.nchannels

	return makeTransformStream([]Stream{input, cutoff, resonance, gain}, func(inputs []Stream) Stepper {
		sInput := inputs[0]
		sCutoff := inputs[1].Mono()
		sResonance := inputs[2].Mono()
		sGain := inputs[3].Mono()

		step := svfStepper(sInput, sCutoff, sResonance, nil)
		gNext := sGain.Next

		out := make(Frame, nchannels)

		return func() (Frame, bool) {
			lp, bp, hp, k, ok := step()
			if !ok {
				return nil, false
			}
			gFrame, ok := gNext()
			if !ok {
				return nil, false
			}

			A := gFrame[0]
			// Reasonable clamp: negative gains are phase inversions and can blow up
			// expectations for an EQ-style control.
			if A < 0 {
				A = 0
			}

			// Neutral response from SVF core: notch = lp + hp.
			for c := range nchannels {
				notch := lp[c] + hp[c]
				out[c] = notch + (A-1)*k*bp[c]
			}
			return out, true
		}
	})
}

// DigitalSVF applies a Vital-inspired digital state-variable filter.
// Parameters are streams to allow modulation:
//
//	input:     audio input (N channels)
//	cutoff:    cutoff frequency in Hz (mono stream)
//	resonance: resonance (Q). Values <= 0 are clamped to a small epsilon.
//	blend:     blend in [-1,1], mapping lowpass(-1) -> bandpass(0) -> highpass(+1).
func DigitalSVF(input, cutoff, resonance, blend Stream) Stream {
	nchannels := input.nchannels

	// Let makeTransformStream compute nframes as the shortest among inputs.
	return makeTransformStream([]Stream{input, cutoff, resonance, blend}, func(inputs []Stream) Stepper {
		sInput := inputs[0]
		sCutoff := inputs[1].Mono()
		sResonance := inputs[2].Mono()
		step := svfStepper(sInput, sCutoff, sResonance, nil)

		sBlend := inputs[3].Mono()
		bNext := sBlend.Next

		out := make(Frame, nchannels)

		return func() (Frame, bool) {
			lp, bp, hp, _, ok := step()
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
	RegisterWord("svf", func(vm *VM) error {
		blend, err := vm.GetStream(":blend")
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
		vm.Push(DigitalSVF(input, cutoff, resonance, blend))
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

	RegisterWord("notch2", func(vm *VM) error {
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
		vm.Push(Notch2(input, cutoff, resonance))
		return nil
	})

	RegisterWord("peak2", func(vm *VM) error {
		gain, err := vm.GetStream(":gain")
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
		vm.Push(Peak2(input, cutoff, resonance, gain))
		return nil
	})
}
