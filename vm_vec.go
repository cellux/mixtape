package main

import (
	"fmt"
)

type Vec []Val

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		arg := Pop[Vec](vm)
		vm.PushVal(len(arg))
		return nil
	})
}
