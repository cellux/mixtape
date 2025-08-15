package main

import (
	"fmt"
)

type Vec []Val

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
			next = nil
		} else {
			next = v[i]
			i++
		}
		vm.Push(next)
		return nil
	}
}

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		v := Pop[Vec](vm)
		vm.Push(len(v))
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
	RegisterMethod[Vec]("map", 2, func(vm *VM) error {
		e := Pop[Evaler](vm)
		v := Top[Vec](vm)
		if len(v) == 0 {
			return nil
		}
		for i, item := range v {
			vm.Push(item)
			e.Eval(vm)
			v[i] = vm.Pop()
		}
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
			e.Eval(vm)
		}
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
