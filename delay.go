package main

func (s Stream) Delay(nframes int) Stream {
	if nframes <= 0 {
		return s.clone()
	}

	return makeDelayedStream(s, nframes, func(input Stream) Stepper {
		out := make(Frame, s.nchannels)
		next := input.Next
		remaining := nframes
		return func() (Frame, bool) {
			if remaining > 0 {
				remaining--
				for i := range out {
					out[i] = 0
				}
				return out, true
			}
			return next()
		}
	})
}

// Z1 returns a one-sample delay with an explicit initial frame.
// The first output frame is the provided init; thereafter each output frame
// is the previous input frame.
func Z1(s Stream, init Frame) Stream {
	nchannels := s.nchannels
	return makeDelayedStream(s, 1, func(input Stream) Stepper {
		snext := input.Next
		prev := make(Frame, nchannels)
		copy(prev, init)
		out := make(Frame, nchannels)
		first := true
		finalSent := false
		return func() (Frame, bool) {
			if first {
				first = false
				copy(out, prev)
				frame, ok := snext()
				if ok {
					copy(prev, frame)
				}
				return out, true
			}
			frame, ok := snext()
			if ok {
				copy(out, prev)
				copy(prev, frame)
				return out, true
			}
			if finalSent {
				return nil, false
			}
			finalSent = true
			copy(out, prev)
			return out, true
		}
	})
}

func init() {
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
}
