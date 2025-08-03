package main

import (
	"fmt"
	"math"
)

type Num float64

func (n Num) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Num:
		return n == rhs
	default:
		return false
	}
}

func init() {
	RegisterMethod[Num]("not", 1, func(vm *VM) error {
		arg := Pop[Num](vm)
		vm.PushVal(arg == 0)
		return nil
	})

	RegisterMethod[Num]("assert", 1, func(vm *VM) error {
		n := Pop[Num](vm)
		if n == False {
			return fmt.Errorf("assertion failed")
		}
		return nil
	})

	RegisterMethod[Num]("+", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs + rhs)
		return nil
	})

	RegisterMethod[Num]("-", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs - rhs)
		return nil
	})

	RegisterMethod[Num]("*", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs * rhs)
		return nil
	})

	RegisterMethod[Num]("/", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs / rhs)
		return nil
	})

	RegisterMethod[Num]("%", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(math.Mod(float64(lhs), float64(rhs)))
		return nil
	})

	RegisterMethod[Num]("<", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs < rhs)
		return nil
	})

	RegisterMethod[Num]("<=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs <= rhs)
		return nil
	})

	RegisterMethod[Num](">=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs >= rhs)
		return nil
	})

	RegisterMethod[Num](">", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs > rhs)
		return nil
	})
}

func (n Num) String() string {
	return fmt.Sprintf("%g", n)
}

func (n Num) Stream() Stream {
	return makeStream(1,
		func(yield func(Frame) bool) {
			out := make([]Smp, 1)
			out[0] = Smp(n)
			for {
				if !yield(out) {
					return
				}
			}
		})
}
