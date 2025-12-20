package main

import (
	"errors"
	"fmt"
)

type ThrowValue struct {
	v Val
}

func (e ThrowValue) Error() string {
	return fmt.Sprintf("%s", e.v)
}

func init() {
	RegisterWord("nil", func(vm *VM) error {
		vm.Push(Nil)
		return nil
	})

	RegisterWord("throw", func(vm *VM) error {
		v := vm.Pop()
		return Err{Pos: vm.currentPos, Err: ThrowValue{v}}
	})

	RegisterWord("catch", func(vm *VM) error {
		body := vm.Pop()
		stackState := vm.SaveStackState()
		err := vm.Eval(body)
		vm.RestoreStackState(stackState)
		if err == nil {
			vm.Push(Nil)
		} else {
			var tv ThrowValue
			if errors.As(err, &tv) {
				vm.Push(tv.v)
				err = nil
			}
		}
		return err
	})

	RegisterWord("loop", func(vm *VM) error {
		body := vm.Pop()
		for {
			stackState := vm.SaveStackState()
			err := vm.Eval(body)
			if err != nil {
				vm.RestoreStackState(stackState)
				var tv ThrowValue
				if errors.As(err, &tv) {
					if tv.v == Nil {
						return nil
					}
				}
				return err
			}
		}
	})

	RegisterWord("log", func(vm *VM) error {
		v := vm.Top()
		logger.Info(fmt.Sprintf("%s", v))
		return nil
	})

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
		return vm.DoQuote()
	})

	RegisterWord("set", func(vm *VM) error {
		v := vm.Pop()
		k := vm.Pop()
		vm.SetVal(k, v)
		return nil
	})

	RegisterWord("get", func(vm *VM) error {
		k := vm.Pop()
		v := vm.GetVal(k)
		vm.Push(v)
		return nil
	})

	RegisterWord("eval", func(vm *VM) error {
		return vm.DoEval()
	})

	RegisterWord("iter", func(vm *VM) error {
		return vm.DoIter()
	})

	RegisterWord("next", func(vm *VM) error {
		return vm.DoNext()
	})
}
