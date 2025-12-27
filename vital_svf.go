package main

import (
	"math"
)

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

// svfMix maps a blend in [-1,1] to low/band/high amounts similar to Vital's default blend mapping.
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

// DigitalSVF applies a Vital-inspired digital state-variable filter.
// Parameters are streams to allow modulation:
//
//	input:     audio input (N channels)
//	cutoff:    cutoff frequency in Hz (mono stream)
//	resonance: resonance (Q). Values <= 0 are clamped to a small epsilon.
//	drive:     input drive multiplier.
//	blend:     blend in [-1,1], mapping lowpass(-1) -> bandpass(0) -> highpass(+1).
//	saturate:  if true, applies tanh saturation to the output.
func DigitalSVF(input, cutoff, resonance, drive, blend Stream, saturate bool) Stream {
	nchannels := input.nchannels

	// Let makeTransformStream compute nframes as the shortest among inputs.
	return makeTransformStream([]Stream{input, cutoff, resonance, drive, blend}, func(inputs []Stream) Stepper {
		inNext := inputs[0].WithNChannels(nchannels).Next
		cNext := inputs[1].Mono().Next
		rNext := inputs[2].Mono().Next
		dNext := inputs[3].Mono().Next
		bNext := inputs[4].Mono().Next

		state := newDigitalSVFState(nchannels)
		out := make(Frame, nchannels)

		return func() (Frame, bool) {
			inFrame, ok := inNext()
			if !ok {
				return nil, false
			}
			cFrame, ok := cNext()
			if !ok {
				return nil, false
			}
			rFrame, ok := rNext()
			if !ok {
				return nil, false
			}
			dFrame, ok := dNext()
			if !ok {
				return nil, false
			}
			bFrame, ok := bNext()
			if !ok {
				return nil, false
			}

			cut := cFrame[0]
			res := rFrame[0]
			drv := dFrame[0]
			blendVal := bFrame[0]
			lowAmt, bandAmt, highAmt := svfMix(blendVal)

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

				lp := v2
				bp := v1
				hp := x - k*bp - lp

				y := lowAmt*lp + bandAmt*bp + highAmt*hp
				if saturate {
					y = math.Tanh(y)
				}
				out[c] = y
			}

			return out, true
		}
	})
}

func init() {
	RegisterWord("vital/svf", func(vm *VM) error {
		satNum, err := vm.GetNum(":saturate")
		if err != nil {
			return err
		}
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
		vm.Push(DigitalSVF(input, cutoff, resonance, drive, blend, satNum != 0))
		return nil
	})
}
