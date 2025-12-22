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

func makeStream(nchannels, nframes int, seq iter.Seq[Frame]) Stream {
	return Stream{
		nchannels: nchannels,
		nframes:   nframes,
		seq:       seq,
	}
}

// makeTransformStream creates a stream which transforms N input streams into a single output stream.
// The output stream:
//   - has the same number of channels as the first input
//   - has nframes = 0 if all inputs are infinite
//     has nframes = length of the shortest input otherwise
func makeTransformStream(inputs []Stream, seq iter.Seq[Frame]) Stream {
	nchannels := inputs[0].nchannels
	nframesMin := inputs[0].nframes
	nframesMax := inputs[0].nframes
	for _, s := range inputs {
		if s.nframes > 0 && s.nframes < nframesMin {
			nframesMin = s.nframes
		}
		if s.nframes > nframesMax {
			nframesMax = s.nframes
		}
	}
	var nframes int
	if nframesMax == 0 {
		nframes = 0
	} else {
		nframes = nframesMin
	}
	return makeStream(nchannels, nframes, seq)
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

func (s Stream) Mono() Stream {
	if s.nchannels == 1 {
		return s
	}
	return makeStream(1, s.nframes, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		for frame := range s.seq {
			var sum Smp
			for ch := range s.nchannels {
				sum += frame[ch]
			}
			out[0] = sum / Smp(len(frame))
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
	return makeStream(2, s.nframes, func(yield func(Frame) bool) {
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

func (s Stream) Combine(other Stream, op SmpBinOp) Stream {
	nchannels := s.nchannels
	return makeTransformStream([]Stream{s, other}, func(yield func(Frame) bool) {
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
}

func (s Stream) Join(other Stream) Stream {
	var nframes int
	if s.nframes > 0 && other.nframes > 0 {
		nframes = s.nframes + other.nframes
	}
	return makeStream(s.nchannels, nframes, func(yield func(Frame) bool) {
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
