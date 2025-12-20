package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type Num float64

func (n Num) implVal() {}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
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

func (n Num) Eval(vm *VM) error {
	vm.Push(n)
	return nil
}

func (n Num) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Num:
		return n == rhs
	default:
		return false
	}
}

func init() {
	RegisterMethod[Num]("if", 2, func(vm *VM) error {
		block := vm.Pop()
		cond := Pop[Num](vm)
		if cond != 0 {
			return vm.Eval(block)
		}
		return nil
	})

	RegisterMethod[Num]("if", 3, func(vm *VM) error {
		elseBlock := vm.Pop()
		ifBlock := vm.Pop()
		cond := Pop[Num](vm)
		if cond != 0 {
			return vm.Eval(ifBlock)
		} else {
			return vm.Eval(elseBlock)
		}
	})

	RegisterMethod[Num]("<", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.Push(lhs < rhs)
		return nil
	})

	RegisterMethod[Num]("<=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.Push(lhs <= rhs)
		return nil
	})

	RegisterMethod[Num](">=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.Push(lhs >= rhs)
		return nil
	})

	RegisterMethod[Num](">", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.Push(lhs > rhs)
		return nil
	})
}

func (n Num) String() string {
	return formatFloat(float64(n))
}

func (n Num) Stream() Stream {
	return makeStream(1, func(yield func(Frame) bool) {
		out := make(Frame, 1)
		out[0] = Smp(n)
		for {
			if !yield(out) {
				return
			}
		}
	})
}
