package main

import (
	"fmt"
)

type Vec []Val

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func (v Vec) Execute(vm *VM) error {
	for _, val := range v {
		if val == Sym("--") && !vm.IsQuoting() {
			break
		}
		err := vm.Execute(val)
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

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		v := Pop[Vec](vm)
		vm.Push(len(v))
		return nil
	})
	RegisterMethod[Vec]("map", 2, func(vm *VM) error {
		e := Pop[Executable](vm)
		v := Top[Vec](vm)
		if len(v) == 0 {
			return nil
		}
		for i, item := range v {
			vm.Push(item)
			e.Execute(vm)
			v[i] = vm.Pop()
		}
		return nil
	})
	RegisterMethod[Vec]("reduce", 2, func(vm *VM) error {
		e := Pop[Executable](vm)
		v := Pop[Vec](vm)
		if len(v) == 0 {
			vm.Push(v)
			return nil
		}
		vm.Push(v[0])
		for i := 1; i < len(v); i++ {
			vm.Push(v[i])
			e.Execute(vm)
		}
		return nil
	})
	RegisterMethod[Vec]("join", 1, func(vm *VM) error {
		vm.Push(Vec{Sym("join")})
		return vm.Execute(Sym("reduce"))
	})
	RegisterMethod[Vec]("+", 1, func(vm *VM) error {
		vm.Push(Vec{Sym("+")})
		return vm.Execute(Sym("reduce"))
	})
	RegisterMethod[Vec]("*", 1, func(vm *VM) error {
		vm.Push(Vec{Sym("*")})
		return vm.Execute(Sym("reduce"))
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
