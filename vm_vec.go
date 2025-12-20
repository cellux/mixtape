package main

import (
	"fmt"
)

type Vec []Val

func (v Vec) implVal() {}

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func (v Vec) Eval(vm *VM) error {
	for _, val := range v {
		if val == Sym("--") && !vm.IsQuoting() {
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

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		v := Pop[Vec](vm)
		vm.Push(len(v))
		return nil
	})
	RegisterMethod[Vec]("at", 2, func(vm *VM) error {
		index := int(Pop[Num](vm))
		v := Pop[Vec](vm)
		if index < 0 || index >= len(v) {
			return fmt.Errorf("at: index out of bounds: %d", index)
		}
		vm.Push(v[index])
		return nil
	})
	RegisterMethod[Vec]("clone", 1, func(vm *VM) error {
		src := Top[Vec](vm)
		dst := make(Vec, len(src))
		copy(dst, src)
		vm.Push(dst)
		return nil
	})
	RegisterMethod[Vec]("push", 2, func(vm *VM) error {
		item := vm.Pop()
		v := Pop[Vec](vm)
		v = append(v, item)
		vm.Push(v)
		return nil
	})
	RegisterMethod[Vec]("pop", 1, func(vm *VM) error {
		v := Pop[Vec](vm)
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
		e := Pop[Evaler](vm)
		v := Pop[Vec](vm)
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
		e := Pop[Evaler](vm)
		v := Pop[Vec](vm)
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
		e := Pop[Evaler](vm)
		v := Pop[Vec](vm)
		if len(v) == 0 {
			vm.Push(v)
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
		step := int(Pop[Num](vm))
		size := int(Pop[Num](vm))
		v := Pop[Vec](vm)
		vm.Push(v.Partition(size, step))
		return nil
	})
	RegisterMethod[Vec]("tape", 1, func(vm *VM) error {
		v := Pop[Vec](vm)
		t := pushTape(vm, 1, len(v))
		for i, item := range v {
			if n, ok := item.(Num); ok {
				t.samples[i] = Smp(n)
			} else {
				return fmt.Errorf("Vec.tape: expected numeric items, got %T", item)
			}
		}
		return nil
	})
}
