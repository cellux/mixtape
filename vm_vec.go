package main

import (
	"fmt"
)

type Vec []Val

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
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
		arg := Pop[Vec](vm)
		vm.PushVal(len(arg))
		return nil
	})
}
