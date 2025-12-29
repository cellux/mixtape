package main

import (
	"math"

	"github.com/dh1tw/gosamplerate"
)

const resampleBlockFrames = 1024

type streamResampler struct {
	src       gosamplerate.Src
	ratio     float64
	nchannels int
	in        Stream
	inBlock   []float32
	outBuf    []Smp
	finished  bool
	closed    bool
}

func (r *streamResampler) appendSamples(samples []float32) {
	for _, smp := range samples {
		r.outBuf = append(r.outBuf, Smp(smp))
	}
}

func (r *streamResampler) close() {
	if r.closed {
		return
	}
	_ = gosamplerate.Delete(r.src)
	r.closed = true
}

func (r *streamResampler) flushTail() {
	for {
		tail, err := r.src.Process(nil, r.ratio, true)
		if err != nil || len(tail) == 0 {
			break
		}
		r.appendSamples(tail)
	}
	r.close()
}

func (r *streamResampler) fillAndProcessBlock(next Stepper) {
	r.inBlock = r.inBlock[:0]
	for len(r.inBlock) < cap(r.inBlock) {
		frame, ok := next()
		if !ok {
			break
		}
		for c := 0; c < r.nchannels; c++ {
			r.inBlock = append(r.inBlock, float32(frame[c]))
		}
	}

	endOfInput := len(r.inBlock) < cap(r.inBlock)
	if len(r.inBlock) == 0 && endOfInput {
		r.finished = true
		r.flushTail()
		return
	}

	processed, err := r.src.Process(r.inBlock, r.ratio, endOfInput)
	if err != nil {
		r.finished = true
		r.flushTail()
		return
	}

	r.appendSamples(processed)
	if endOfInput {
		r.finished = true
		r.flushTail()
	}
}

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

		r := &streamResampler{
			src:       src,
			ratio:     ratio,
			nchannels: nchannels,
			inBlock:   make([]float32, 0, resampleBlockFrames*nchannels),
			outBuf:    make([]Smp, 0, resampleBlockFrames*nchannels*2),
		}

		next := in.Next
		frame := make(Frame, nchannels)

		return func() (Frame, bool) {
			for len(r.outBuf) < nchannels {
				if r.finished {
					if len(r.outBuf) == 0 {
						return nil, false
					}
					break
				}
				r.fillAndProcessBlock(next)
			}

			if len(r.outBuf) < nchannels {
				return nil, false
			}

			copy(frame, r.outBuf[:nchannels])
			r.outBuf = r.outBuf[nchannels:]

			if r.finished && len(r.outBuf) == 0 {
				r.close()
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
