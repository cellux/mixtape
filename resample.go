package main

import (
	"math"

	"github.com/dh1tw/gosamplerate"
)

const (
	resampleBlockFrames = 1024
	resampleMaxRatio    = 1.0 * 16
	resampleMinRatio    = 1.0 / 16
)

func resampleStream(input Stream, converterType int, ratio float64) Stream {
	nchannels := input.nchannels

	if input.nframes > 0 {
		// one-shot case
		t := input.Take(nil, input.nframes)
		tempBuf := make([]float32, t.nframes*t.nchannels)
		for i, smp := range t.samples {
			tempBuf[i] = float32(smp)
		}
		resampledBuf, err := gosamplerate.Simple(tempBuf, ratio, t.nchannels, converterType)
		if err != nil {
			return makeEmptyStream(nchannels)
		}
		resampledFrames := len(resampledBuf) / nchannels
		return makeRewindableStream(nchannels, resampledFrames, func() Stepper {
			readIndex := 0
			frame := make(Frame, nchannels)
			return func() (Frame, bool) {
				if readIndex == len(resampledBuf) {
					return nil, false
				}
				for ch := range nchannels {
					frame[ch] = Smp(resampledBuf[readIndex])
					readIndex++
				}
				return frame, true
			}
		})
	}

	// streaming case
	return makeRewindableStream(nchannels, 0, func() Stepper {
		inputBufferLen := resampleBlockFrames * nchannels
		outputBufferLen := int(math.Ceil(resampleBlockFrames*resampleMaxRatio)) * nchannels
		src, err := gosamplerate.New(converterType, nchannels, outputBufferLen)
		if err != nil {
			return func() (Frame, bool) { return nil, false }
		}

		inBlock := make([]float32, 0, inputBufferLen)
		var outBlock []float32

		next := input.clone().Next
		frame := make(Frame, nchannels)

		outIndex := 0
		endOfInput := false

		return func() (Frame, bool) {
			for {
				if outBlock != nil && outIndex < len(outBlock) {
					for ch := range nchannels {
						frame[ch] = Smp(outBlock[outIndex])
						outIndex++
					}
					return frame, true
				}
				if endOfInput {
					return nil, false
				}
				outBlock = nil
				outIndex = 0
				inBlock = inBlock[:0]
				writeIndex := 0
				for writeIndex < cap(inBlock) {
					frame, ok := next()
					if !ok {
						endOfInput = true
						break
					}
					for ch := range nchannels {
						inBlock = append(inBlock, float32(frame[ch]))
						writeIndex++
					}
				}
				if writeIndex == 0 {
					return nil, false
				}
				outBlock, err = src.Process(inBlock, ratio, endOfInput)
				if err != nil {
					endOfInput = true
					return nil, false
				}
			}
		}
	})
}

func isValidRatio(ratio float64) bool {
	if !gosamplerate.IsValidRatio(ratio) {
		return false
	}
	if ratio < resampleMinRatio || ratio > resampleMaxRatio {
		return false
	}
	return true
}

func init() {
	RegisterMethod[Streamable]("resample", 2, func(vm *VM) error {
		ratioNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		ratio := float64(ratioNum)
		if !isValidRatio(ratio) {
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
		if !isValidRatio(ratio) {
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
