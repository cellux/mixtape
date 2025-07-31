package main

import (
	"fmt"
	"path/filepath"
	"regexp"
)

type Str string

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
		vm.PushVal(f)
		return nil
	})

	RegisterMethod[Str]("=", 2, func(vm *VM) error {
		rhs := Pop[Str](vm)
		lhs := Pop[Str](vm)
		vm.PushVal(lhs == rhs)
		return nil
	})

	RegisterMethod[Str]("path/join", 2, func(vm *VM) error {
		rhs := Pop[Str](vm)
		lhs := Pop[Str](vm)
		vm.PushVal(filepath.Join(string(lhs), string(rhs)))
		return nil
	})
}

func (s Str) String() string {
	return string(s)
}
