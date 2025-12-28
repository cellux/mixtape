package main

import (
	"fmt"
	"math"
)

func Phasor(freq Stream, phase float64) Stream {
	return makeRewindableStream(1, 0, func() Stepper {
		fnext := freq.Mono().Next
		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		p := Smp(phase)
		sr := Smp(SampleRate())
		out := make(Frame, 1)
		return func() (Frame, bool) {
			f, ok := fnext()
			if !ok {
				return nil, false
			}
			out[0] = p
			periodSamples := sr / f[0]
			if periodSamples == 0 {
				return nil, false
			}
			incr := 1.0 / periodSamples
			p = math.Mod(p+incr, 1.0)
			return out, true
		}
	})
}

// impulseStream produces a mono infinite stream of impulses (value 1) at the
// provided frequency. Output is 0 elsewhere. Phase is in [0,1).
func impulseStream(freq Stream, phase float64) Stream {
	return makeRewindableStream(1, 0, func() Stepper {
		fnext := freq.Mono().Next
		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		p := Smp(phase)
		sr := Smp(SampleRate())
		out := make(Frame, 1)
		return func() (Frame, bool) {
			f, ok := fnext()
			if !ok {
				return nil, false
			}

			inc := f[0] / sr
			if inc <= 0 {
				out[0] = 0
			} else {
				p += inc
				if p >= 1 {
					p = math.Mod(p, 1.0)
					out[0] = 1
				} else {
					out[0] = 0
				}
			}

			return out, true
		}
	})
}

// Peak computes the maximum absolute value per frame, returning a mono stream.
func Peak(s Stream) Stream {
	return makeRewindableStream(1, s.nframes, func() Stepper {
		out := make(Frame, 1)
		next := s.clone().Next
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			maxAbs := Smp(0)
			for c := range s.nchannels {
				v := frame[c]
				if v < 0 {
					v = -v
				}
				if v > maxAbs {
					maxAbs = v
				}
			}
			out[0] = maxAbs
			return out, true
		}
	})
}

func (s Stream) Skip(nframes int) Stream {
	return makeTransformStream([]Stream{s}, func(inputs []Stream) Stepper {
		next := inputs[0].Next
		skipped := 0
		return func() (Frame, bool) {
			for skipped < nframes {
				_, ok := next()
				if !ok {
					return nil, false
				}
				skipped++
			}
			return next()
		}
	})
}

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

// SampleHold implements sample & hold: latches input on each rate wrap.
func SampleHold(input Stream, rate Stream) Stream {
	nchannels := input.nchannels
	sr := Smp(SampleRate())

	return makeTransformStream([]Stream{input, rate}, func(inputs []Stream) Stepper {
		out := make(Frame, nchannels)
		held := make(Frame, nchannels)
		inext := inputs[0].Next
		rnext := inputs[1].Mono().Next
		hasHeld := false
		p := Smp(0)
		return func() (Frame, bool) {
			rframe, ok := rnext()
			if !ok {
				return nil, false
			}
			frame, ok := inext()
			if !ok {
				return nil, false
			}

			inc := rframe[0] / sr
			if inc < 0 {
				inc = 0
			}

			if !hasHeld {
				copy(held, frame)
				hasHeld = true
			}

			p += inc
			if p >= 1 {
				p = math.Mod(p, 1.0)
				copy(held, frame)
			}

			copy(out, held)
			return out, true
		}
	})
}

// Pan applies equal-power panning to a mono stream, returning stereo.
// Pan value can be a Num or Streamable providing values in [-1..1].
func Pan(s Stream, pan Stream) Stream {
	return makeTransformStream([]Stream{s, pan}, func(inputs []Stream) Stepper {
		snext := inputs[0].Mono().Next
		pnext := inputs[1].Mono().Next
		out := make(Frame, 2)
		return func() (Frame, bool) {
			sframe, ok := snext()
			if !ok {
				return nil, false
			}
			pframe, ok := pnext()
			if !ok {
				return nil, false
			}
			l, r := equalPowerPan(float64(pframe[0]))
			out[0] = sframe[0] * Smp(l)
			out[1] = sframe[0] * Smp(r)
			return out, true
		}
	})
}

// Mix takes N streams and mixes two neighbors into a single stream.
//
// The mixed streams are selected by the ratio argument. A value of
// 0.7 with N=2 results in a 30%/70% mix of the first and second
// streams.
func Mix(ss []Stream, ratio Stream) Stream {
	nchannels := ss[0].nchannels
	allStreams := append(ss[:], ratio.Mono())
	return makeTransformStream(allStreams, func(inputs []Stream) Stepper {
		nexts := make([]Stepper, len(inputs))
		frames := make([]Frame, len(inputs)-1)
		ratioIndex := len(inputs) - 1
		for i, s := range inputs {
			nexts[i] = s.Next
		}
		out := make(Frame, nchannels)
		return func() (Frame, bool) {
			for ch := range nchannels {
				out[ch] = 0
			}
			ratioFrame, ok := nexts[ratioIndex]()
			if !ok {
				return nil, false
			}
			ratio := ratioFrame[0]
			if ratio > 1 {
				ratio = 1
			}
			if ratio < 0 {
				ratio = 0
			}
			n := len(ss)
			floatIndex := ratio * Smp(n-1)
			leftIndex := min(n-1, int(floatIndex))
			rightIndex := min(n-1, leftIndex+1)
			rightWeight := floatIndex - Smp(leftIndex)
			leftWeight := 1.0 - rightWeight
			for i := range ss {
				frames[i], ok = nexts[i]()
				if !ok {
					return nil, false
				}
			}
			lframe := frames[leftIndex]
			rframe := frames[rightIndex]
			for ch := range nchannels {
				out[ch] = lframe[ch]*leftWeight + rframe[ch]*rightWeight
			}
			return out, true
		}
	})
}

func init() {
	RegisterWord("~phasor", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase, err := vm.GetFloat(":phase")
		if err != nil {
			return err
		}
		vm.Push(Phasor(freq, phase))
		return nil
	})

	RegisterWord("~impulse", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}

		phase := 0.0
		if pval := vm.GetVal(":phase"); pval != nil {
			if pnum, ok := pval.(Num); ok {
				phase = float64(pnum)
			} else {
				return fmt.Errorf("impulse: :phase must be number")
			}
		}

		vm.Push(impulseStream(freq, phase))
		return nil
	})

	RegisterWord("sh", func(vm *VM) error {
		// input rate -- output
		rate, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		input, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(SampleHold(input, rate))
		return nil
	})

	RegisterWord("peak", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(Peak(stream))
		return nil
	})

	RegisterWord("softclip", func(vm *VM) error {
		nfMode, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		mode := int(nfMode)
		switch mode {
		case 0: // tanh
			return applySmpUnOp(vm, TanhOp())
		case 1: // atan (scaled to [-1,1])
			return applySmpUnOp(vm, func(x Smp) Smp {
				return (2.0 / math.Pi) * math.Atan(x)
			})
		case 2: // cubic soft clip
			return applySmpUnOp(vm, func(x Smp) Smp {
				if x < -1 {
					return -2.0 / 3.0
				}
				if x > 1 {
					return 2.0 / 3.0
				}
				return x - (x*x*x)/3.0
			})
		case 3: // softsign
			return applySmpUnOp(vm, func(x Smp) Smp {
				return x / (1 + math.Abs(x))
			})
		default:
			return vm.Errorf("softclip: invalid mode (%d)", mode)
		}
	})

	RegisterWord("skip", func(vm *VM) error {
		nfNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		nf := int(nfNum)
		if nf <= 0 {
			return vm.Errorf("skip: nframes must be positive")
		}
		vm.Push(stream.Skip(nf))
		return nil
	})

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

	RegisterWord("mix", func(vm *VM) error {
		// inputs ratio -- output
		ratio, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		inputs, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		if len(inputs) == 0 {
			return vm.Errorf("mix: input vec is empty")
		}
		if len(inputs) == 1 {
			vm.Push(inputs[0])
			return nil
		}
		streams := make([]Stream, len(inputs))
		for i, v := range inputs {
			s, err := streamFromVal(v)
			if err != nil {
				return err
			}
			streams[i] = s
		}
		nchannels := streams[0].nchannels
		for _, s := range streams {
			if s.nchannels != nchannels {
				return vm.Errorf("mix: all inputs must have the same number of channels")
			}
		}
		vm.Push(Mix(streams, ratio))
		return nil
	})
}
