package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type Str string

func (s Str) Execute(vm *VM) error {
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

var floatRegex = regexp.MustCompile(`^[0-9_eE./+-]+`)

func scanFloat(text string) (float64, error) {
	var f float64
	match := floatRegex.FindString(text)
	if match == "" {
		return 0, fmt.Errorf("cannot parse float: %s", text)
	}
	_, err := fmt.Sscanf(match, "%g", &f)
	if err == nil {
		var nominator, denominator int
		_, err = fmt.Sscanf(match, "%d/%d", &nominator, &denominator)
		if err == nil {
			return float64(nominator) / float64(denominator), nil
		} else {
			return f, nil
		}
	} else {
		return 0, fmt.Errorf("cannot parse float: %s", text)
	}
}

func init() {
	RegisterMethod[Str]("num", 1, func(vm *VM) error {
		arg := Pop[Str](vm)
		f, err := scanFloat(string(arg))
		if err != nil {
			return err
		}
		vm.Push(f)
		return nil
	})

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
}

func (s Str) String() string {
	return string(s)
}
