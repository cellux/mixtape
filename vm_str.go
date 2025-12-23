package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Str string

func (s Str) getVal() Val { return s }

func (s Str) String() string {
	return string(s)
}

func (s Str) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Str:
		return s == rhs
	default:
		return false
	}
}

func init() {
	RegisterMethod[Str]("sym", 1, func(vm *VM) error {
		s, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		vm.Push(Sym(s))
		return nil
	})

	RegisterMethod[Str]("+", 2, func(vm *VM) error {
		rhs, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		lhs, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		vm.Push(lhs + rhs)
		return nil
	})

	RegisterMethod[Str]("path/join", 2, func(vm *VM) error {
		rhs, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		lhs, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		vm.Push(filepath.Join(string(lhs), string(rhs)))
		return nil
	})

	RegisterMethod[Str]("parse", 1, func(vm *VM) error {
		s, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		code, err := vm.Parse(strings.NewReader(string(s)), "<string>")
		if err != nil {
			return err
		}
		vm.Push(code)
		return nil
	})

	RegisterMethod[Str]("parse1", 1, func(vm *VM) error {
		if err := vm.Eval(Sym("parse")); err != nil {
			return err
		}
		v, err := Pop[Vec](vm)
		if err != nil {
			return err
		}
		if len(v) == 0 {
			return fmt.Errorf("parse1: empty string")
		}
		vm.Push(v[0])
		return nil
	})
}
