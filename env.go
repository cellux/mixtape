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
	RegisterWord("/line", func(vm *VM) error {
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		return nil
	})

	RegisterWord("/exp", func(vm *VM) error {
		kNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		k := float64(kNum)
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		if k == 0 {
			vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		} else {
			vm.Push(envseg(start, end, nf, func(x float64) float64 {
				return (math.Exp(k*x) - 1) / (math.Exp(k) - 1)
			}))
		}
		return nil
	})

	RegisterWord("/log", func(vm *VM) error {
		kNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		k := float64(kNum)
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		if k == 0 {
			vm.Push(envseg(start, end, nf, func(x float64) float64 { return x }))
		} else {
			vm.Push(envseg(start, end, nf, func(x float64) float64 {
				return math.Log(1+k*x) / math.Log(1+k)
			}))
		}
		return nil
	})

	RegisterWord("/cos", func(vm *VM) error {
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		vm.Push(envseg(start, end, nf, func(x float64) float64 {
			return 0.5 - 0.5*math.Cos(math.Pi*x)
		}))
		return nil
	})

	RegisterWord("/pow", func(vm *VM) error {
		pNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		p := float64(pNum)
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		vm.Push(envseg(start, end, nf, func(x float64) float64 { return math.Pow(x, p) }))
		return nil
	})

	RegisterWord("/sigmoid", func(vm *VM) error {
		kNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		startNum, err := vm.GetNum(":start")
		if err != nil {
			return err
		}
		endNum, err := vm.GetNum(":end")
		if err != nil {
			return err
		}
		nfNum, err := vm.GetNum(":nf")
		if err != nil {
			return err
		}
		k := float64(kNum)
		start := float64(startNum)
		end := float64(endNum)
		nf := int(nfNum)
		vm.Push(envseg(start, end, nf, func(x float64) float64 {
			return 1 / (1 + math.Exp(-k*(x-0.5)))
		}))
		return nil
	})
}
