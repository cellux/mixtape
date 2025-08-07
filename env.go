package main

import (
	"math"
)

func envseg(start, end float64, nframes int, shape func(float64) float64) *Tape {
	t := makeTape(1, nframes)
	var x float64
	incr := 1.0 / float64(nframes)
	for i := range nframes {
		t.samples[i] = start + float64(end-start)*shape(x)
		x += incr
	}
	return t
}

func init() {
	RegisterWord("line/", func(vm *VM) error {
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		return nil
	})

	RegisterWord("exp/", func(vm *VM) error {
		k := float64(Pop[Num](vm))
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		if k == 0 {
			vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		} else {
			vm.Push(envseg(start, end, nf, func(x float64) float64 {
				return (math.Exp(k*x) - 1) / (math.Exp(k) - 1)
			}))
		}
		return nil
	})

	RegisterWord("log/", func(vm *VM) error {
		k := float64(Pop[Num](vm))
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		if k == 0 {
			vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		} else {
			vm.Push(envseg(start, end, nf, func(x float64) float64 {
				return math.Log(1+k*x) / math.Log(1+k)
			}))
		}
		return nil
	})

	RegisterWord("cos/", func(vm *VM) error {
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		vm.Push(envseg(start, end, nf, func(x float64) float64 {
			return 0.5 - 0.5*math.Cos(math.Pi*x)
		}))
		return nil
	})

	RegisterWord("pow/", func(vm *VM) error {
		p := float64(Pop[Num](vm))
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		vm.Push(envseg(start, end, nf, func(x float64) float64 { return math.Pow(x, p) }))
		return nil
	})

	RegisterWord("sigmoid/", func(vm *VM) error {
		k := float64(Pop[Num](vm))
		start := float64(Get[Num](vm, ":start"))
		end := float64(Get[Num](vm, ":end"))
		nf := int(Get[Num](vm, ":nf"))
		vm.Push(envseg(start, end, nf, func(x float64) float64 {
			return 1 / (1 + math.Exp(-k*(x-0.5)))
		}))
		return nil
	})

}
