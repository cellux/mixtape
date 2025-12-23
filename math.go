package main

import (
	"math"
	"math/rand"
)

var rng *rand.Rand

func init() {
	source := rand.NewSource(0)
	rng = rand.New(source)
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

func SinOp() SmpUnOp {
	return func(x Smp) Smp { return math.Sin(x) }
}

func CosOp() SmpUnOp {
	return func(x Smp) Smp { return math.Cos(x) }
}

func TanOp() SmpUnOp {
	return func(x Smp) Smp { return math.Tan(x) }
}

func AsinOp() SmpUnOp {
	return func(x Smp) Smp { return math.Asin(x) }
}

func AcosOp() SmpUnOp {
	return func(x Smp) Smp { return math.Acos(x) }
}

func AtanOp() SmpUnOp {
	return func(x Smp) Smp { return math.Atan(x) }
}

func SinhOp() SmpUnOp {
	return func(x Smp) Smp { return math.Sinh(x) }
}

func CoshOp() SmpUnOp {
	return func(x Smp) Smp { return math.Cosh(x) }
}

func TanhOp() SmpUnOp {
	return func(x Smp) Smp { return math.Tanh(x) }
}

func AsinhOp() SmpUnOp {
	return func(x Smp) Smp { return math.Asinh(x) }
}

func AcoshOp() SmpUnOp {
	return func(x Smp) Smp { return math.Acosh(x) }
}

func AtanhOp() SmpUnOp {
	return func(x Smp) Smp { return math.Atanh(x) }
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

func Atan2Op() SmpBinOp {
	return func(y, x Smp) Smp { return math.Atan2(float64(y), float64(x)) }
}

func HypotOp() SmpBinOp {
	return func(x, y Smp) Smp { return math.Hypot(float64(x), float64(y)) }
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

	RegisterWord("sin", func(vm *VM) error {
		return applySmpUnOp(vm, SinOp())
	})

	RegisterWord("cos", func(vm *VM) error {
		return applySmpUnOp(vm, CosOp())
	})

	RegisterWord("tan", func(vm *VM) error {
		return applySmpUnOp(vm, TanOp())
	})

	RegisterWord("asin", func(vm *VM) error {
		return applySmpUnOp(vm, AsinOp())
	})

	RegisterWord("acos", func(vm *VM) error {
		return applySmpUnOp(vm, AcosOp())
	})

	RegisterWord("atan", func(vm *VM) error {
		return applySmpUnOp(vm, AtanOp())
	})

	RegisterWord("sinh", func(vm *VM) error {
		return applySmpUnOp(vm, SinhOp())
	})

	RegisterWord("cosh", func(vm *VM) error {
		return applySmpUnOp(vm, CoshOp())
	})

	RegisterWord("tanh", func(vm *VM) error {
		return applySmpUnOp(vm, TanhOp())
	})

	RegisterWord("asinh", func(vm *VM) error {
		return applySmpUnOp(vm, AsinhOp())
	})

	RegisterWord("acosh", func(vm *VM) error {
		return applySmpUnOp(vm, AcoshOp())
	})

	RegisterWord("atanh", func(vm *VM) error {
		return applySmpUnOp(vm, AtanhOp())
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

	RegisterWord("atan2", func(vm *VM) error {
		return applySmpBinOp(vm, Atan2Op())
	})

	RegisterWord("hypot", func(vm *VM) error {
		return applySmpBinOp(vm, HypotOp())
	})

	RegisterWord("min", func(vm *VM) error {
		return applySmpBinOp(vm, MinOp())
	})

	RegisterWord("max", func(vm *VM) error {
		return applySmpBinOp(vm, MaxOp())
	})

	RegisterWord("rand", func(vm *VM) error {
		// result is in range [0.0,1.0)
		vm.Push(Num(rng.Float64()))
		return nil
	})

	RegisterWord("rand/seed", func(vm *VM) error {
		seedNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		rng.Seed(int64(seedNum))
		return nil
	})

}
