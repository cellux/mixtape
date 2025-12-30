package main

import (
	"fmt"
	"iter"
)

type Stepper func() (Frame, bool)
type StepperFactory func() Stepper

type Stream struct {
	nchannels  int
	nframes    int
	newStepper StepperFactory
	next       Stepper
}

func (s Stream) getVal() Val { return s }

func (s Stream) String() string {
	return fmt.Sprintf("Stream(%d,%d)", s.nchannels, s.nframes)
}

// Next returns the next frame and whether it is valid.
func (s Stream) Next() (Frame, bool) {
	if s.next == nil {
		return nil, false
	}
	return s.next()
}

// clone returns a fresh Stream with its own Stepper, if available.
func (s Stream) clone() Stream {
	if s.newStepper == nil {
		return s
	}
	return Stream{
		nchannels:  s.nchannels,
		nframes:    s.nframes,
		newStepper: s.newStepper,
		next:       s.newStepper(),
	}
}

// Seq exposes the stream as an iter.Seq to keep range-style iteration without goroutines.
func (s Stream) Seq() iter.Seq[Frame] {
	return func(yield func(Frame) bool) {
		for {
			f, ok := s.Next()
			if !ok {
				return
			}
			if !yield(f) {
				return
			}
		}
	}
}

type Streamable interface {
	Val
	Stream() Stream
}

func makeStream(nchannels, nframes int, next Stepper) Stream {
	return Stream{
		nchannels: nchannels,
		nframes:   nframes,
		next:      next,
	}
}

// makeRewindableStream constructs a Stream whose iteration can be restarted
// by cloning. The factory must produce an independent Stepper each time.
func makeRewindableStream(nchannels, nframes int, factory StepperFactory) Stream {
	return Stream{
		nchannels:  nchannels,
		nframes:    nframes,
		newStepper: factory,
		next:       factory(),
	}
}

// makeDelayedStream constructs a stream that conceptually prepends `extraFrames`
// frames before the (possibly finite) input stream.
//
// For finite inputs (nframes > 0), the result length is input.nframes + extraFrames.
// For inputs with unknown/infinite length (nframes == 0), the result length is left
// as 0.
//
// The provided factory receives a cloned input stream and must implement the
// actual delay behavior.
func makeDelayedStream(input Stream, extraFrames int, factory func(Stream) Stepper) Stream {
	if extraFrames < 0 {
		extraFrames = 0
	}

	nframes := 0
	if input.nframes > 0 {
		nframes = input.nframes + extraFrames
	}

	return makeRewindableStream(input.nchannels, nframes, func() Stepper {
		return factory(input.clone())
	})
}

// makeTransformStream creates a stream which transforms N input streams into a single output stream.
// The output stream:
//   - has the same number of channels as the first input
//   - has nframes = 0 if all inputs are infinite
//     has nframes = length of the shortest finite input otherwise
func makeTransformStream(inputs []Stream, mk func([]Stream) Stepper) Stream {
	nchannels := inputs[0].nchannels
	nframesMin := inputs[0].nframes
	nframesMax := inputs[0].nframes

	for _, s := range inputs {
		if s.nframes > 0 && (nframesMin == 0 || s.nframes < nframesMin) {
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

	return makeRewindableStream(nchannels, nframes, func() Stepper {
		clones := make([]Stream, len(inputs))
		for i, s := range inputs {
			clones[i] = s.clone()
		}
		return mk(clones)
	})
}

func makeEmptyStream(nchannels int) Stream {
	return makeStream(nchannels, 0, func() (Frame, bool) {
		return nil, false
	})
}

func streamFromVal(v Val) (Stream, error) {
	if s, ok := v.(Streamable); ok {
		return s.Stream(), nil
	}
	return Stream{}, fmt.Errorf("expected streamable value, got %T", v)
}

func (s Stream) Stream() Stream {
	return s
}

func (s Stream) Take(vm *VM, nframes int) *Tape {
	nchannels := s.nchannels
	t := makeTape(nchannels, nframes)
	writeIndex := 0
	end := nframes * nchannels
	pct1 := end / 100
	pct1 = pct1 - (pct1 % nchannels)
	for frame := range s.Seq() {
		for ch := range nchannels {
			t.samples[writeIndex] = frame[ch]
			writeIndex++
		}
		if writeIndex == end {
			break
		}
		if vm != nil {
			// Check cancellation frequently enough to make C-g feel responsive,
			// but only report progress occasionally.
			if vm.CancelRequested() {
				break
			}
			if pct1 > 0 && writeIndex%pct1 == 0 {
				vm.ReportTapeProgress(t, end/nchannels, writeIndex/nchannels)
			}
		}
	}
	return t
}

func (s Stream) Frames(vm *VM) (Vec, error) {
	if s.nframes == 0 {
		return nil, vm.Errorf("frames: attempt to turn infinite stream into finite vec")
	}
	v := make(Vec, 0, s.nframes)
	for frame := range s.Seq() {
		if s.nchannels == 1 {
			v = append(v, Num(frame[0]))
		} else {
			sv := make(Vec, s.nchannels)
			for ch, smp := range frame {
				sv[ch] = Num(smp)
			}
			v = append(v, sv)
		}
	}
	return v, nil
}

func (s Stream) Mono() Stream {
	if s.nchannels == 1 {
		return s.clone()
	}
	return makeRewindableStream(1, s.nframes, func() Stepper {
		out := make(Frame, 1)
		next := s.clone().Next
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			var sum Smp
			for ch := range s.nchannels {
				sum += frame[ch]
			}
			out[0] = sum / Smp(len(frame))
			return out, true
		}
	})
}

func (s Stream) Stereo() Stream {
	if s.nchannels == 2 {
		return s.clone()
	}
	return makeRewindableStream(2, s.nframes, func() Stepper {
		out := make(Frame, 2)
		next := s.clone().Next
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			out[0] = frame[0]
			out[1] = frame[0]
			return out, true
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
	return makeTransformStream([]Stream{s, other}, func(inputs []Stream) Stepper {
		out := make(Frame, nchannels)
		lhs := inputs[0]
		rhs := inputs[1]
		snext := lhs.WithNChannels(nchannels).Next
		onext := rhs.WithNChannels(nchannels).Next
		return func() (Frame, bool) {
			frame, ok := snext()
			if !ok {
				return nil, false
			}
			oframe, ok := onext()
			if !ok {
				return nil, false
			}
			for i := range nchannels {
				out[i] = op(frame[i], oframe[i])
			}
			return out, true
		}
	})
}

func (s Stream) Join(other Stream) Stream {
	var nframes int
	if s.nframes > 0 && other.nframes > 0 {
		nframes = s.nframes + other.nframes
	}
	return makeRewindableStream(s.nchannels, nframes, func() Stepper {
		// Each consumer gets its own traversal; reset the steppers per clone.
		lhs := s.clone()
		rhs := other.clone()
		snext := lhs.Next
		onext := rhs.Next
		phase := 0
		return func() (Frame, bool) {
			if phase == 0 {
				frame, ok := snext()
				if ok {
					return frame, true
				}
				phase = 1
			}
			return onext()
		}
	})
}

func applySmpUnOp(vm *VM, op SmpUnOp) error {
	input, err := Pop[Streamable](vm)
	if err != nil {
		return err
	}
	if n, ok := input.(Num); ok {
		vm.Push(op(Smp(n)))
		return nil
	}
	s := input.Stream()
	result := makeTransformStream([]Stream{s}, func(inputs []Stream) Stepper {
		s := inputs[0]
		out := make(Frame, s.nchannels)
		next := s.Next
		return func() (Frame, bool) {
			frame, ok := next()
			if !ok {
				return nil, false
			}
			for ch := range s.nchannels {
				out[ch] = op(frame[ch])
			}
			return out, true
		}
	})
	vm.Push(result)
	return nil
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
	RegisterWord("~empty", func(vm *VM) error {
		nchannelsNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		nchannels := int(nchannelsNum)
		if nchannels < 1 {
			return vm.Errorf("~empty: invalid number of channels: %d", int(nchannelsNum))
		}
		vm.Push(makeEmptyStream(nchannels))
		return nil
	})

	RegisterWord("~", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream)
		return nil
	})

	RegisterMethod[Streamable]("len", 1, func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream.nframes)
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
		vm.Push(stream.Take(vm, int(nfNum)))
		return nil
	})

	RegisterWord("frames", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vec, err := stream.Frames(vm)
		if err != nil {
			return err
		}
		vm.Push(vec)
		return nil
	})

	RegisterWord("mono", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream.Mono())
		return nil
	})

	RegisterWord("stereo", func(vm *VM) error {
		stream, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		vm.Push(stream.Stereo())
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

}
