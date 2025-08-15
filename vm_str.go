package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Str string

func (s Str) Eval(vm *VM) error {
	vm.Push(s)
	return nil
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
	RegisterMethod[Str]("path/join", 2, func(vm *VM) error {
		rhs := Pop[Str](vm)
		lhs := Pop[Str](vm)
		vm.Push(filepath.Join(string(lhs), string(rhs)))
		return nil
	})

	RegisterMethod[Str]("parse", 1, func(vm *VM) error {
		arg := Pop[Str](vm)
		code, err := vm.Parse(strings.NewReader(string(arg)), "<string>")
		if err != nil {
			return err
		}
		vm.Push(code)
		return nil
	})

	RegisterMethod[Str]("parse1", 1, func(vm *VM) error {
		vm.Eval(Sym("parse"))
		v := Pop[Vec](vm)
		if len(v) == 0 {
			return fmt.Errorf("parse1: empty string")
		}
		vm.Push(v[0])
		return nil
	})
}

func (s Str) String() string {
	return string(s)
}
