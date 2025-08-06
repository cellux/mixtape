package main

import (
	"fmt"
)

type Sym string

func (s Sym) Execute(vm *VM) error {
	name := string(s)
	if name[0] == ':' {
		vm.Push(vm.GetVal(name))
		return nil
	}
	method := vm.FindMethod(name)
	if method != nil {
		return method(vm)
	}
	word := vm.GetVal(name)
	if word != nil {
		return vm.Execute(word)
	}
	return fmt.Errorf("word or method not found: %s", name)
}

func (s Sym) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Sym:
		return s == rhs
	default:
		return false
	}
}
