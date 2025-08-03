package main

import (
	"fmt"
)

func init() {
	RegisterWord("sr", func(vm *VM) error {
		vm.Push(SampleRate())
		return nil
	})

	RegisterWord("=", func(vm *VM) error {
		stacksize := len(vm.valStack)
		if stacksize < 2 {
			return fmt.Errorf("=: stack underflow")
		}
		rhs := vm.Pop()
		lhs := vm.Pop()
		vm.Push(Equal(lhs, rhs))
		return nil
	})

	RegisterWord("!=", func(vm *VM) error {
		if err := vm.Execute(Sym("=")); err != nil {
			return err
		}
		result := Pop[Num](vm)
		vm.Push(result == 0)
		return nil
	})

	RegisterWord("stack", func(vm *VM) error {
		vm.Push(vm.valStack)
		return nil
	})

	RegisterWord("str", func(vm *VM) error {
		vm.Push(fmt.Sprintf("%s", vm.Pop()))
		return nil
	})

	RegisterWord("drop", func(vm *VM) error {
		return vm.DoDrop()
	})

	RegisterWord("nip", func(vm *VM) error {
		return vm.DoNip()
	})

	RegisterWord("dup", func(vm *VM) error {
		return vm.DoDup()
	})

	RegisterWord("swap", func(vm *VM) error {
		return vm.DoSwap()
	})

	RegisterWord("over", func(vm *VM) error {
		return vm.DoOver()
	})

	RegisterWord("(", func(vm *VM) error {
		return vm.DoPushEnv()
	})

	RegisterWord(")", func(vm *VM) error {
		return vm.DoPopEnv()
	})

	RegisterWord("[", func(vm *VM) error {
		return vm.DoMark()
	})

	RegisterWord("]", func(vm *VM) error {
		return vm.DoCollect()
	})

	RegisterWord("{", func(vm *VM) error {
		vm.compileBuffer = make(Vec, 0, 64)
		vm.compileLevel = 1
		return nil
	})

	RegisterWord("set", func(vm *VM) error {
		k := vm.Pop()
		v := vm.Pop()
		vm.SetVal(k, v)
		return nil
	})

	RegisterWord("get", func(vm *VM) error {
		k := vm.Pop()
		v := vm.GetVal(k)
		vm.Push(v)
		return nil
	})

	RegisterWord("do", func(vm *VM) error {
		return vm.DoDo()
	})
}
