package main

import (
	"math"

	"github.com/dh1tw/gosamplerate"
)

const resampleBlockFrames = 1024

func resampleStream(input Stream, converterType int, ratio float64) Stream {
	nchannels := input.nchannels
	nframes := 0
	if input.nframes > 0 {
		nframes = int(math.Ceil(float64(input.nframes) * ratio))
	}

	return makeRewindableStream(nchannels, nframes, func() Stepper {
		in := input.clone()
		src, err := gosamplerate.New(converterType, nchannels, resampleBlockFrames)
		if err != nil {
			return func() (Frame, bool) { return nil, false }
		}

		inBlock := make([]float32, 0, resampleBlockFrames*nchannels)
		outBuf := make([]Smp, 0, resampleBlockFrames*nchannels*2)
		finished := false
		deleted := false

		flush := func() {
			for {
				tail, err := src.Process(nil, ratio, true)
				if err != nil {
					break
				}
				if len(tail) == 0 {
					break
				}
				for _, smp := range tail {
					outBuf = append(outBuf, Smp(smp))
				}
			}
			if !deleted {
				_ = gosamplerate.Delete(src)
				deleted = true
			}
		}

		frame := make(Frame, nchannels)

		return func() (Frame, bool) {
			for len(outBuf) < nchannels {
				if finished {
					if len(outBuf) == 0 {
						return nil, false
					}
					break
				}

				inBlock = inBlock[:0]
				for len(inBlock) < cap(inBlock) {
					frame, ok := in.Next()
					if !ok {
						break
					}
					for c := 0; c < nchannels; c++ {
						inBlock = append(inBlock, float32(frame[c]))
					}
				}

				endOfInput := len(inBlock) < cap(inBlock)
				if len(inBlock) == 0 && endOfInput {
					finished = true
					flush()
					break
				}

				processed, err := src.Process(inBlock, ratio, endOfInput)
				if err != nil {
					finished = true
					flush()
					break
				}

				for _, smp := range processed {
					outBuf = append(outBuf, Smp(smp))
				}

				if endOfInput {
					finished = true
					flush()
				}
			}

			if len(outBuf) < nchannels {
				return nil, false
			}

			copy(frame, outBuf[:nchannels])
			outBuf = outBuf[nchannels:]

			if finished && len(outBuf) == 0 && !deleted {
				_ = gosamplerate.Delete(src)
				deleted = true
			}

			return frame, true
		}
	})
}

func init() {
	RegisterMethod[Streamable]("resample", 2, func(vm *VM) error {
		ratioNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		ratio := float64(ratioNum)
		if ratio <= 0 {
			return vm.Errorf("resample: invalid ratio: %f", ratio)
		}
		converterType, err := vm.GetInt(":resample/converter")
		if err != nil {
			return err
		}
		if converterType < 0 || converterType > 4 {
			return vm.Errorf("resample: invalid converterType in :resample/converter: %d - must be between 0..4", converterType)
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(resampleStream(stream, converterType, ratio))
		return nil
	})

	RegisterMethod[*Tape]("resample", 2, func(vm *VM) error {
		ratioNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		ratio := float64(ratioNum)
		if ratio <= 0 {
			return vm.Errorf("resample: invalid ratio: %f", ratio)
		}
		converterType, err := vm.GetInt(":resample/converter")
		if err != nil {
			return err
		}
		if converterType < 0 || converterType > 4 {
			return vm.Errorf("resample: invalid converterType: %d - must be between 0..4", converterType)
		}
		t, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		tempBuf := make([]float32, t.nframes*t.nchannels)
		for i, smp := range t.samples {
			tempBuf[i] = float32(smp)
		}
		resampledBuf, err := gosamplerate.Simple(tempBuf, ratio, t.nchannels, converterType)
		if err != nil {
			return err
		}
		resampledFrames := len(resampledBuf) / t.nchannels
		resampledTape := pushTape(vm, t.nchannels, resampledFrames)
		for i, smp := range resampledBuf {
			resampledTape.samples[i] = Smp(smp)
		}
		return nil
	})
}
