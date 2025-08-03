package main

import (
	"fmt"
	"iter"
	"math"
)

type Stream struct {
	nchannels int
	seq       iter.Seq[Frame]
}

type Streamable interface {
	Stream() Stream
}

func makeStream(nchannels int, seq iter.Seq[Frame]) Stream {
	return Stream{
		nchannels: nchannels,
		seq:       seq,
	}
}

func SinOp() SmpUnOp {
	return func(phase Smp) Smp {
		return math.Sin(phase * 2 * math.Pi)
	}
}

func PulseOp(width float64) SmpUnOp {
	return func(phase Smp) Smp {
		if phase < width {
			return 1.0
		} else {
			return -1.0
		}
	}
}

func TriangleOp() SmpUnOp {
	return func(phase Smp) Smp {
		if phase < 0.25 {
			return phase * 4.0
		} else if phase < 0.75 {
			return 1.0 - (phase-0.25)*4.0
		} else {
			return -1.0 + (phase-0.75)*4.0
		}
	}
}

func SawOp() SmpUnOp {
	return func(phase Smp) Smp {
		if phase < 0.5 {
			return phase * 2.0
		} else {
			return -1.0 + (phase-0.5)*2.0
		}
	}
}

func Phasor(freq Stream, op SmpUnOp) Stream {
	return makeStream(1,
		func(yield func(Frame) bool) {
			out := make([]Smp, 1)
			fnext, fstop := iter.Pull(freq.Mono().seq)
			defer fstop()
			phase := Smp(0)
			sr := Smp(SampleRate())
			for {
				out[0] = op(phase)
				if !yield(out) {
					return
				}
				f, ok := fnext()
				if !ok {
					return
				}
				periodSamples := sr / f[0]
				incr := 1.0 / periodSamples
				phase = math.Mod(phase+incr, 1.0)
			}
		})
}

func (s Stream) Stream() Stream {
	return s
}

func (s Stream) Take(nframes int) *Tape {
	nchannels := s.nchannels
	t := makeTape(nchannels, nframes)
	writeIndex := 0
	end := nframes * nchannels
	for frame := range s.seq {
		for i := range nchannels {
			t.samples[writeIndex] = frame[i]
			writeIndex++
		}
		if writeIndex == end {
			break
		}
	}
	return t
}

func (s Stream) Delay(nframes int) Stream {
	nchannels := s.nchannels
	return makeStream(nchannels,
		func(yield func(Frame) bool) {
			out := make([]Smp, nchannels)
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

func (s Stream) Mono() Stream {
	if s.nchannels == 1 {
		return s
	}
	return makeStream(1,
		func(yield func(Frame) bool) {
			out := make([]Smp, 1)
			for frame := range s.seq {
				out[0] = (frame[0] + frame[1]) / 2.0
				if !yield(out) {
					return
				}
			}
		})
}

func (s Stream) Stereo() Stream {
	if s.nchannels == 2 {
		return s
	}
	return makeStream(2,
		func(yield func(Frame) bool) {
			out := make([]Smp, 2)
			for frame := range s.seq {
				out[0] = frame[0]
				out[1] = frame[0]
				if !yield(out) {
					return
				}
			}
		})
}

func (s Stream) AdaptChannels(other Stream) Stream {
	switch other.nchannels {
	case 1:
		return s.Mono()
	case 2:
		return s.Stereo()
	}
	return s
}

func (s Stream) Combine(other Stream, op SmpBinOp) Stream {
	nchannels := s.nchannels
	return makeStream(nchannels,
		func(yield func(Frame) bool) {
			out := make([]Smp, nchannels)
			onext, ostop := iter.Pull(other.AdaptChannels(s).seq)
			defer ostop()
			for frame := range s.seq {
				oframe, ok := onext()
				if !ok {
					return
				}
				for i := range nchannels {
					out[i] = op(frame[i], oframe[i])
				}
				if !yield(out) {
					return
				}
			}
		})
}

func applySmpBinOp(vm *VM, op SmpBinOp) error {
	top := vm.Pop()
	rhs, ok := top.(Streamable)
	if !ok {
		return fmt.Errorf("object of type %T does not implement Streamable", top)
	}
	top = vm.Pop()
	lhs, ok := top.(Streamable)
	if !ok {
		return fmt.Errorf("object of type %T does not implement Streamable", top)
	}
	if n1, n1ok := lhs.(Num); n1ok {
		if n2, n2ok := rhs.(Num); n2ok {
			vm.Push(op(Smp(n1), Smp(n2)))
			return nil
		}
	}
	result := lhs.Stream().Combine(rhs.Stream(), op)
	if t, ok := lhs.(*Tape); ok {
		vm.Push(result.Take(t.nframes))
	} else if t, ok := rhs.(*Tape); ok {
		vm.Push(result.Take(t.nframes))
	} else {
		vm.Push(result)
	}
	return nil
}

func init() {
	RegisterWord("~", func(vm *VM) error {
		top := vm.Pop()
		streamable, ok := top.(Streamable)
		if !ok {
			return fmt.Errorf("object of type %T does not implement Streamable", top)
		}
		vm.Push(streamable.Stream())
		return nil
	})

	RegisterWord("sin~", func(vm *VM) error {
		freq := vm.GetStream(":freq")
		vm.Push(Phasor(freq, SinOp()))
		return nil
	})

	RegisterWord("saw~", func(vm *VM) error {
		freq := vm.GetStream(":freq")
		vm.Push(Phasor(freq, SawOp()))
		return nil
	})

	RegisterWord("triangle~", func(vm *VM) error {
		freq := vm.GetStream(":freq")
		vm.Push(Phasor(freq, TriangleOp()))
		return nil
	})

	RegisterWord("pulse~", func(vm *VM) error {
		freq := vm.GetStream(":freq")
		width := vm.GetFloat(":width")
		vm.Push(Phasor(freq, PulseOp(width)))
		return nil
	})

	RegisterWord("take", func(vm *VM) error {
		nf := int(Pop[Num](vm))
		s := Pop[Stream](vm)
		vm.Push(s.Take(nf))
		return nil
	})

	RegisterWord("delay", func(vm *VM) error {
		nf := int(Pop[Num](vm))
		s := Pop[Stream](vm)
		vm.Push(s.Delay(nf))
		return nil
	})

	RegisterWord("+", func(vm *VM) error {
		return applySmpBinOp(vm, func(x, y Smp) Smp { return x + y })
	})

	RegisterWord("-", func(vm *VM) error {
		return applySmpBinOp(vm, func(x, y Smp) Smp { return x - y })
	})

	RegisterWord("*", func(vm *VM) error {
		return applySmpBinOp(vm, func(x, y Smp) Smp { return x * y })
	})

	RegisterWord("/", func(vm *VM) error {
		return applySmpBinOp(vm, func(x, y Smp) Smp { return x / y })
	})

	RegisterWord("%", func(vm *VM) error {
		return applySmpBinOp(vm, func(x, y Smp) Smp { return math.Mod(float64(x), float64(y)) })
	})
}
