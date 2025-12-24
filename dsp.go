package main

import (
	"fmt"
	"iter"
	"math"
)

// impulseStream produces a mono infinite stream of impulses (value 1) at the
// provided frequency. Output is 0 elsewhere. Phase is in [0,1).
func impulseStream(freq Stream, phase float64) Stream {
	return makeStream(1, 0, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		fnext, fstop := iter.Pull(freq.Mono().seq)
		defer fstop()

		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		p := Smp(phase)
		sr := Smp(SampleRate())

		for {
			f, ok := fnext()
			if !ok {
				return
			}

			inc := f[0] / sr
			if inc <= 0 {
				out[0] = 0
			} else {
				p += inc
				if p >= 1 {
					p = Smp(math.Mod(float64(p), 1.0))
					out[0] = 1
				} else {
					out[0] = 0
				}
			}

			if !yield(out) {
				return
			}
		}
	})
}

// DCMeanCenter subtracts the per-channel mean for finite streams.
func DCMeanCenter(s Stream) Stream {
	t := s.Take(nil, s.nframes)

	sums := make([]float64, t.nchannels)
	readIndex := 0
	for range t.nframes {
		for ch := range t.nchannels {
			sums[ch] += t.samples[readIndex]
			readIndex++
		}
	}

	means := make([]Smp, t.nchannels)
	for ch := range t.nchannels {
		mean := sums[ch] / float64(t.nframes)
		if math.Abs(mean) < 1e-12 {
			mean = 0
		}
		means[ch] = Smp(mean)
	}

	writeIndex := 0
	for range t.nframes {
		for ch := range t.nchannels {
			t.samples[writeIndex] -= means[ch]
			writeIndex++
		}
	}

	return t.Stream()
}

// DCBlock applies a simple one-pole high-pass filter to remove DC offset.
// alpha controls the cutoff; typical small value like 0.995.
func DCBlock(s Stream, alpha float64) Stream {
	if s.nframes != 0 {
		return DCMeanCenter(s)
	}
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

func (s Stream) Delay(nframes int) Stream {
	return makeTransformStream([]Stream{s}, func(yield func(Frame) bool) {
		out := make(Frame, s.nchannels)
		for range nframes {
			if !yield(out) {
				return
			}
		}
		for frame := range s.seq {
			if !yield(frame) {
				return
			}
		}
	})
}

// Z1 returns a one-sample delay with an explicit initial frame.
// The first output frame is the provided init; thereafter each output frame
// is the previous input frame.
func Z1(s Stream, init Frame) Stream {
	var nframes int
	if s.nframes == 0 {
		nframes = 0
	} else {
		nframes = s.nframes + 1
	}
	nchannels := s.nchannels
	return makeStream(nchannels, nframes, func(yield func(Frame) bool) {
		prev := make(Frame, nchannels)
		copy(prev, init)
		out := make(Frame, nchannels)
		for frame := range s.seq {
			copy(out, prev)
			copy(prev, frame)
			if !yield(out) {
				return
			}
		}
		if !yield(prev) {
			return
		}
	})
}

func (s Stream) Skip(nframes int) Stream {
	return makeTransformStream([]Stream{s}, func(yield func(Frame) bool) {
		skipped := 0
		for frame := range s.seq {
			if skipped < nframes {
				skipped++
				continue
			}
			if !yield(frame) {
				return
			}
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

	return makeTransformStream([]Stream{input, rate}, func(yield func(Frame) bool) {
		out := make(Frame, nchannels)
		held := make(Frame, nchannels)
		hasHeld := false
		p := Smp(0)

		rnext, rstop := iter.Pull(rate.Mono().seq)
		defer rstop()

		for frame := range input.seq {
			rframe, ok := rnext()
			if !ok {
				return
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
				p = Smp(math.Mod(float64(p), 1.0))
				copy(held, frame)
			}

			copy(out, held)
			if !yield(out) {
				return
			}
		}
	})
}

// Pan applies equal-power panning to a mono stream, returning stereo.
// Pan value can be a Num or Streamable providing values in [-1..1].
func Pan(s Stream, pan Stream) Stream {
	return makeStream(2, s.nframes, func(yield func(Frame) bool) {
		out := make(Frame, 2)
		snext, sstop := iter.Pull(s.Mono().seq)
		pnext, pstop := iter.Pull(pan.Mono().seq)
		defer sstop()
		defer pstop()
		for {
			sframe, ok := snext()
			if !ok {
				return
			}
			pframe, ok := pnext()
			if !ok {
				return
			}
			l, r := equalPowerPan(float64(pframe[0]))
			out[0] = sframe[0] * Smp(l)
			out[1] = sframe[0] * Smp(r)
			if !yield(out) {
				return
			}
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
	ratioIndex := len(ss)
	return makeTransformStream(allStreams, func(yield func(Frame) bool) {
		out := make(Frame, nchannels)
		nexts := make([]func() (Frame, bool), len(allStreams))
		stops := make([]func(), len(allStreams))
		frames := make([]Frame, len(ss))
		for i, s := range allStreams {
			next, stop := iter.Pull(s.seq)
			nexts[i] = next
			stops[i] = stop
		}
		defer func() {
			for _, stop := range stops {
				stop()
			}
		}()
		for {
			for ch := range nchannels {
				out[ch] = 0
			}
			ratioFrame, ok := nexts[ratioIndex]()
			if !ok {
				return
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
					return
				}
			}
			lframe := frames[leftIndex]
			rframe := frames[rightIndex]
			for ch := range nchannels {
				out[ch] = lframe[ch]*leftWeight + rframe[ch]*rightWeight
			}
			if !yield(out) {
				return
			}
		}
	})
}

func init() {
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

	RegisterWord("dc", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(DCFilter(stream))
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
				return Smp((2.0 / math.Pi) * math.Atan(float64(x)))
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
				return x / (1 + Smp(math.Abs(float64(x))))
			})
		default:
			return vm.Errorf("softclip: invalid mode (%d)", mode)
		}
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

	RegisterWord("delay", func(vm *VM) error {
		nfNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream.Delay(int(nfNum)))
		return nil
	})

	RegisterWord("z1*", func(vm *VM) error {
		initVal := vm.Pop()
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}

		nchannels := stream.nchannels
		initFrame := make(Frame, nchannels)

		switch v := initVal.(type) {
		case Num:
			for i := range nchannels {
				initFrame[i] = Smp(v)
			}
		case Vec:
			if len(v) != nchannels {
				return vm.Errorf("z1*: init vec length must match channel count (got %d, expected %d)", len(v), nchannels)
			}
			for i, item := range v {
				num, ok := item.(Num)
				if !ok {
					return vm.Errorf("z1*: init vec items must be numbers (index %d has %T)", i, item)
				}
				initFrame[i] = Smp(num)
			}
		default:
			return vm.Errorf("z1*: init must be Num or Vec, got %T", initVal)
		}

		vm.Push(Z1(stream, initFrame))
		return nil
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
