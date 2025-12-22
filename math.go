package main

import (
	"math"
	"math/rand"
)

func getRNG(vm *VM) (*rand.Rand, error) {
	seedNum, err := vm.GetNum(":seed")
	if err != nil {
		return nil, err
	}
	source := rand.NewSource(int64(seedNum))
	rng := rand.New(source)
	return rng, nil
}

func AbsOp() SmpUnOp {
	return func(x Smp) Smp { return math.Abs(x) }
}

func ExpOp() SmpUnOp {
	return func(x Smp) Smp { return math.Exp(x) }
}

func Exp2Op() SmpUnOp {
	return func(x Smp) Smp { return math.Exp2(x) }
}

func Log10Op() SmpUnOp {
	return func(x Smp) Smp { return math.Log10(x) }
}

func Log2Op() SmpUnOp {
	return func(x Smp) Smp { return math.Log2(x) }
}

func FloorOp() SmpUnOp {
	return func(x Smp) Smp { return math.Floor(x) }
}

func CeilOp() SmpUnOp {
	return func(x Smp) Smp { return math.Ceil(x) }
}

func TruncOp() SmpUnOp {
	return func(x Smp) Smp { return math.Trunc(x) }
}

func RoundOp() SmpUnOp {
	return func(x Smp) Smp { return math.Round(x) }
}

func AddOp() SmpBinOp {
	return func(x, y Smp) Smp { return x + y }
}

func SubOp() SmpBinOp {
	return func(x, y Smp) Smp { return x - y }
}

func MulOp() SmpBinOp {
	return func(x, y Smp) Smp { return x * y }
}

func DivOp() SmpBinOp {
	return func(x, y Smp) Smp { return x / y }
}

func ModOp() SmpBinOp {
	return func(x, y Smp) Smp { return math.Mod(float64(x), float64(y)) }
}

func RemOp() SmpBinOp {
	return func(x, y Smp) Smp { return math.Remainder(float64(x), float64(y)) }
}

func PowOp() SmpBinOp {
	return func(x, y Smp) Smp { return math.Pow(float64(x), float64(y)) }
}

func MinOp() SmpBinOp {
	return func(x, y Smp) Smp { return min(x, y) }
}

func MaxOp() SmpBinOp {
	return func(x, y Smp) Smp { return max(x, y) }
}

func init() {

	RegisterWord("e", func(vm *VM) error {
		vm.Push(math.E)
		return nil
	})

	RegisterWord("pi", func(vm *VM) error {
		vm.Push(math.Pi)
		return nil
	})

	RegisterWord("abs", func(vm *VM) error {
		return applySmpUnOp(vm, AbsOp())
	})

	RegisterWord("exp", func(vm *VM) error {
		return applySmpUnOp(vm, ExpOp())
	})

	RegisterWord("exp2", func(vm *VM) error {
		return applySmpUnOp(vm, Exp2Op())
	})

	RegisterWord("log10", func(vm *VM) error {
		return applySmpUnOp(vm, Log10Op())
	})

	RegisterWord("log2", func(vm *VM) error {
		return applySmpUnOp(vm, Log2Op())
	})

	RegisterWord("floor", func(vm *VM) error {
		return applySmpUnOp(vm, FloorOp())
	})

	RegisterWord("ceil", func(vm *VM) error {
		return applySmpUnOp(vm, CeilOp())
	})

	RegisterWord("trunc", func(vm *VM) error {
		return applySmpUnOp(vm, TruncOp())
	})

	RegisterWord("round", func(vm *VM) error {
		return applySmpUnOp(vm, RoundOp())
	})

	RegisterWord("+", func(vm *VM) error {
		return applySmpBinOp(vm, AddOp())
	})

	RegisterWord("-", func(vm *VM) error {
		return applySmpBinOp(vm, SubOp())
	})

	RegisterWord("*", func(vm *VM) error {
		return applySmpBinOp(vm, MulOp())
	})

	RegisterWord("/", func(vm *VM) error {
		return applySmpBinOp(vm, DivOp())
	})

	RegisterWord("mod", func(vm *VM) error {
		return applySmpBinOp(vm, ModOp())
	})

	RegisterWord("rem", func(vm *VM) error {
		return applySmpBinOp(vm, RemOp())
	})

	RegisterWord("pow", func(vm *VM) error {
		return applySmpBinOp(vm, PowOp())
	})

	RegisterWord("min", func(vm *VM) error {
		return applySmpBinOp(vm, MinOp())
	})

	RegisterWord("max", func(vm *VM) error {
		return applySmpBinOp(vm, MaxOp())
	})

	RegisterWord("rand", func(vm *VM) error {
		rng, err := getRNG(vm)
		if err != nil {
			return err
		}
		// result is in range [0.0,1.0)
		vm.Push(Num(rng.Float64()))
		return nil
	})

}
