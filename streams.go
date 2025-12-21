package main

import (
	"fmt"
	"iter"
	"math"
)

type Stream struct {
	nchannels int
	nframes   int
	seq       iter.Seq[Frame]
}

func (s Stream) getVal() Val { return s }

func (s Stream) String() string {
	return fmt.Sprintf("Stream(%d,%d)", s.nchannels, s.nframes)
}

type Streamable interface {
	Val
	Stream() Stream
}

func makeStream(nchannels int, seq iter.Seq[Frame]) Stream {
	return Stream{
		nchannels: nchannels,
		seq:       seq,
	}
}

func makeFiniteStream(nchannels, nframes int, seq iter.Seq[Frame]) Stream {
	return Stream{
		nchannels: nchannels,
		nframes:   nframes,
		seq:       seq,
	}
}

func makeTransformStream(s Stream, seq iter.Seq[Frame]) Stream {
	return Stream{
		nchannels: s.nchannels,
		nframes:   s.nframes,
		seq:       seq,
	}
}

func streamFromVal(v Val) (Stream, error) {
	if v == nil {
		return Num(0).Stream(), nil
	}
	if s, ok := v.(Streamable); ok {
		return s.Stream(), nil
	}
	return Stream{}, fmt.Errorf("expected streamable value, got %T", v)
}

func SinOp() SmpUnOp {
	return func(phase Smp) Smp {
		return math.Sin(phase * 2 * math.Pi)
	}
}

func PulseOp(pw float64) SmpUnOp {
	return func(phase Smp) Smp {
		if phase < pw {
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

func AddOp() SmpBinOp {
	return func(x, y Smp) Smp { return x + y }
}

func SubOp() SmpBinOp {
	return func(x, y Smp) Smp { return x - y }
}

func MulOp() SmpBinOp {
	return func(x, y Smp) Smp { return x * y }
}

func DivOp() SmpBinOp {
	return func(x, y Smp) Smp { return x / y }
}

func ModOp() SmpBinOp {
	return func(x, y Smp) Smp { return math.Mod(float64(x), float64(y)) }
}

func Phasor(freq Stream, phase float64, op SmpUnOp) Stream {
	return makeStream(1, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		fnext, fstop := iter.Pull(freq.Mono().seq)
		defer fstop()
		if phase < 0.0 || phase >= 1.0 {
			phase = 0.0
		}
		phase := Smp(phase)
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
	return makeTransformStream(s, func(yield func(Frame) bool) {
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

func (s Stream) Mono() Stream {
	if s.nchannels == 1 {
		return s
	}
	return makeStream(1, func(yield func(Frame) bool) {
		out := make(Frame, 1)
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
	return makeStream(2, func(yield func(Frame) bool) {
		out := make(Frame, 2)
		for frame := range s.seq {
			out[0] = frame[0]
			out[1] = frame[0]
			if !yield(out) {
				return
			}
		}
	})
}

func (s Stream) WithNChannels(nchannels int) Stream {
	switch nchannels {
	case 1:
		return s.Mono()
	case 2:
		return s.Stereo()
	}
	return s
}

// DCBlock applies a simple one-pole high-pass filter to remove DC offset.
// alpha controls the cutoff; typical small value like 0.995.
func DCBlock(s Stream, alpha float64) Stream {
	return makeTransformStream(s, func(yield func(Frame) bool) {
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

func (s Stream) Combine(other Stream, op SmpBinOp) Stream {
	nchannels := s.nchannels
	result := makeStream(nchannels, func(yield func(Frame) bool) {
		out := make(Frame, nchannels)
		onext, ostop := iter.Pull(other.WithNChannels(nchannels).seq)
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
	if s.nframes > 0 || other.nframes > 0 {
		if s.nframes == 0 {
			result.nframes = other.nframes
		} else if other.nframes == 0 {
			result.nframes = s.nframes
		} else {
			result.nframes = min(s.nframes, other.nframes)
		}
	}
	return result
}

func (s Stream) Join(other Stream) Stream {
	nchannels := s.nchannels
	result := makeStream(nchannels, func(yield func(Frame) bool) {
		for frame := range s.seq {
			if !yield(frame) {
				return
			}
		}
		for frame := range other.seq {
			if !yield(frame) {
				return
			}
		}
	})
	if s.nframes > 0 && other.nframes > 0 {
		result.nframes = s.nframes + other.nframes
	}
	return result
}

func applySmpBinOp(vm *VM, op SmpBinOp) error {
	rhs, err := Pop[Streamable](vm)
	if err != nil {
		return err
	}
	lhs, err := Pop[Streamable](vm)
	if err != nil {
		return err
	}
	if n1, ok := lhs.(Num); ok {
		if n2, ok := rhs.(Num); ok {
			vm.Push(op(Smp(n1), Smp(n2)))
			return nil
		}
	}
	result := lhs.Stream().Combine(rhs.Stream(), op)
	vm.Push(result)
	return nil
}

func init() {
	RegisterWord("~", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream)
		return nil
	})

	RegisterWord("~sin", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase, err := vm.GetFloat(":phase")
		if err != nil {
			return err
		}
		vm.Push(Phasor(freq, phase, SinOp()))
		return nil
	})

	RegisterWord("~saw", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase, err := vm.GetFloat(":phase")
		if err != nil {
			return err
		}
		vm.Push(Phasor(freq, phase, SawOp()))
		return nil
	})

	RegisterWord("~triangle", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase, err := vm.GetFloat(":phase")
		if err != nil {
			return err
		}
		vm.Push(Phasor(freq, phase, TriangleOp()))
		return nil
	})

	RegisterWord("~pulse", func(vm *VM) error {
		freq, err := vm.GetStream(":freq")
		if err != nil {
			return err
		}
		phase, err := vm.GetFloat(":phase")
		if err != nil {
			return err
		}
		pw, err := vm.GetFloat(":pw")
		if err != nil {
			return err
		}
		vm.Push(Phasor(freq, phase, PulseOp(pw)))
		return nil
	})

	RegisterWord("dcblock", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		alpha := 0.995
		if aval := vm.GetVal(":alpha"); aval != nil {
			if anum, ok := aval.(Num); ok {
				alpha = float64(anum)
			} else {
				return fmt.Errorf("dcblock: :alpha must be number")
			}
		}
		vm.Push(DCBlock(stream, alpha))
		return nil
	})

	RegisterWord("take", func(vm *VM) error {
		nfNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream.Take(int(nfNum)))
		return nil
	})

	RegisterMethod[Streamable]("join", 2, func(vm *VM) error {
		rhsStream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		lhsStream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(lhsStream.Join(rhsStream))
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

	RegisterWord("+", func(vm *VM) error {
		return applySmpBinOp(vm, AddOp())
	})

	RegisterWord("-", func(vm *VM) error {
		return applySmpBinOp(vm, SubOp())
	})

	RegisterWord("*", func(vm *VM) error {
		return applySmpBinOp(vm, MulOp())
	})

	RegisterWord("/", func(vm *VM) error {
		return applySmpBinOp(vm, DivOp())
	})

	RegisterWord("%", func(vm *VM) error {
		return applySmpBinOp(vm, ModOp())
	})
}
