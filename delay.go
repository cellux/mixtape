package main

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
}
