package main

import (
	"fmt"
)

type Iterable interface {
	Iter() Fun
}

func init() {
	RegisterWord("seq", func(vm *VM) error {
		body := Pop[Vec](vm)
		syms := Pop[Vec](vm)
		symIterators := make([]Fun, len(syms))
		for i, val := range syms {
			sym, ok := val.(Sym)
			if !ok {
				return fmt.Errorf("seq: syms vec contains value of type %T", val)
			}
			if err := sym.Eval(vm); err != nil {
				return err
			}
			symIterable := vm.Pop()
			if iterable, ok := symIterable.(Iterable); ok {
				symIterators[i] = iterable.Iter()
			} else {
				return fmt.Errorf("seq: value of sym %s is not iterable: %T", sym, symIterable)
			}
		}
		for {
			for i, val := range syms {
				if err := symIterators[i].Eval(vm); err != nil {
					return err
				}
				next := vm.Pop()
				if next == nil {
					return nil
				}
				sym := val.(Sym)
				vm.SetVal(string(sym), next)
			}
			if err := body.Eval(vm); err != nil {
				return err
			}
		}
	})
}
