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
			sym.Execute(vm)
			symIterable := vm.Pop()
			if iterable, ok := symIterable.(Iterable); ok {
				symIterators[i] = iterable.Iter()
			} else {
				return fmt.Errorf("seq: value of sym %s is not iterable: %T", sym, symIterable)
			}
		}
		for {
			for i, val := range syms {
				symIterators[i].Execute(vm)
				next := vm.Pop()
				if next == nil {
					return nil
				}
				sym := val.(Sym)
				vm.SetVal(string(sym), next)
			}
			body.Execute(vm)
		}
	})
}
