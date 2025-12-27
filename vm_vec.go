package main

import (
	"fmt"
)

type Vec []Val

func (v Vec) getVal() Val { return v }

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func (v Vec) allNums() bool {
	for _, item := range v {
		if _, ok := item.(Num); !ok {
			return false
		}
	}
	return true
}

func (v Vec) Stream() Stream {
	if v.allNums() {
		return makeRewindableStream(1, len(v), func() Stepper {
			out := make(Frame, 1)
			index := 0
			return func() (Frame, bool) {
				if index >= len(v) {
					return nil, false
				}
				out[0] = Smp(v[index].(Num))
				index++
				return out, true
			}
		})
	}
	nchannels := 0
	for _, item := range v {
		if sv, ok := item.(Vec); ok {
			if !sv.allNums() {
				return makeEmptyStream(1)
			}
			if nchannels == 0 {
				nchannels = len(sv)
			} else if len(sv) != nchannels {
				return makeEmptyStream(1)
			}
		} else {
			return makeEmptyStream(1)
		}
	}
	return makeRewindableStream(nchannels, len(v), func() Stepper {
		out := make(Frame, nchannels)
		index := 0
		return func() (Frame, bool) {
			if index >= len(v) {
				return nil, false
			}
			sv := v[index].(Vec)
			for ch := range nchannels {
				out[ch] = Smp(sv[ch].(Num))
			}
			index++
			return out, true
		}
	})
}

func (v Vec) Eval(vm *VM) error {
	for _, val := range v {
		if val.getVal() == Sym("--") && !vm.IsQuoting() {
			break
		}
		err := vm.Eval(val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v Vec) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Vec:
		if len(v) != len(rhs) {
			return false
		}
		for index, item := range v {
			if !Equal(item, rhs[index]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (v Vec) Iter() Fun {
	i := 0
	return func(vm *VM) error {
		var next Val
		if i == len(v) {
			next = Nil
		} else {
			next = v[i]
			i++
		}
		vm.Push(next)
		return nil
	}
}

func (v Vec) Partition(size, step int) Vec {
	var out Vec
	for i := 0; i+size <= len(v); i += step {
		out = append(out, v[i:i+size])
	}
	return out
}

// Treat a flat numeric vector as a single-channel tape.
func (v Vec) Tape() *Tape {
	t := makeTape(1, len(v))
	for i, item := range v {
		if n, ok := item.(Num); ok {
			t.samples[i] = Smp(n)
		}
	}
	return t
}

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		vm.Push(len(v))
		return nil
	})
	RegisterMethod[Vec]("at", 2, func(vm *VM) error {
		indexNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		index := int(indexNum)
		if index < 0 || index >= len(v) {
			return fmt.Errorf("at: index out of bounds: %d", index)
		}
		vm.Push(v[index])
		return nil
	})
	RegisterMethod[Vec]("clone", 1, func(vm *VM) error {
		src, err := Top[Vec](vm)
		if err != nil {
			return err
		}
		dst := make(Vec, len(src))
		copy(dst, src)
		vm.Push(dst)
		return nil
	})
	RegisterMethod[Vec]("push", 2, func(vm *VM) error {
		item := vm.Pop()
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		v = append(v, item)
		vm.Push(v)
		return nil
	})
	RegisterMethod[Vec]("pop", 1, func(vm *VM) error {
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		if len(v) == 0 {
			return fmt.Errorf("pop: empty vec")
		}
		item := v[len(v)-1]
		v = v[:len(v)-1]
		vm.Push(v)
		vm.Push(item)
		return nil
	})
	RegisterMethod[Vec]("each", 2, func(vm *VM) error {
		e, err := Pop[Evaler](vm)
		if err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		if len(v) == 0 {
			return nil
		}
		for _, item := range v {
			vm.Push(item)
			if err := e.Eval(vm); err != nil {
				return err
			}
		}
		return nil
	})
	RegisterMethod[Vec]("map", 2, func(vm *VM) error {
		e, err := Pop[Evaler](vm)
		if err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		mapped := make(Vec, 0, len(v))
		for _, item := range v {
			vm.Push(item)
			if err := e.Eval(vm); err != nil {
				return err
			}
			mapped = append(mapped, vm.Pop())
		}
		vm.Push(mapped)
		return nil
	})
	RegisterMethod[Vec]("reduce", 2, func(vm *VM) error {
		e, err := Pop[Evaler](vm)
		if err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		if len(v) == 0 {
			vm.Push(Nil)
			return nil
		}
		vm.Push(v[0])
		for i := 1; i < len(v); i++ {
			vm.Push(v[i])
			if err := e.Eval(vm); err != nil {
				return err
			}
		}
		return nil
	})
	RegisterMethod[Vec]("partition", 3, func(vm *VM) error {
		stepNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		sizeNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		step := int(stepNum)
		size := int(sizeNum)
		vm.Push(v.Partition(size, step))
		return nil
	})
	RegisterWord("vdup", func(vm *VM) error {
		countNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		val := vm.Pop()
		count := int(countNum)
		v := make(Vec, count)
		for i := range count {
			v[i] = val
		}
		vm.Push(v)
		return nil
	})
}
