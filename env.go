package main

import (
	"fmt"
	"math"
)

func validateNF(word string, nf int) error {
	if nf <= 0 {
		return fmt.Errorf("%s: :nf must be > 0 (got %d)", word, nf)
	}
	return nil
}

func validateFinite(word, name string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%s: %s must be finite (got %v)", word, name, v)
	}
	return nil
}

func validateExpK(word string, k float64) error {
	const maxAbsK = 700.0 // below log(MaxFloat64) to avoid exp overflow
	if k > maxAbsK {
		return fmt.Errorf("%s: k must be <= %g to avoid overflow (got %v)", word, maxAbsK, k)
	}
	if k < -maxAbsK {
		return fmt.Errorf("%s: k must be >= %g to avoid overflow (got %v)", word, -maxAbsK, k)
	}
	return nil
}

func validateSigmoidK(word string, k float64) error {
	const maxAbsK = 1400.0 // because exp sees k/2 in worst case
	if k > maxAbsK {
		return fmt.Errorf("%s: k must be <= %g to avoid overflow (got %v)", word, maxAbsK, k)
	}
	if k < -maxAbsK {
		return fmt.Errorf("%s: k must be >= %g to avoid overflow (got %v)", word, -maxAbsK, k)
	}
	return nil
}

func validatePowP(word string, p float64) error {
	if p < 0 {
		return fmt.Errorf("%s: p must be >= 0 to avoid pow singularity at x=0 (got %v)", word, p)
	}
	return nil
}

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
		if err := validateNF("/line", nf); err != nil {
			return err
		}
		if err := validateFinite("/line", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/line", ":end", end); err != nil {
			return err
		}
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
		if err := validateNF("/exp", nf); err != nil {
			return err
		}
		if err := validateFinite("/exp", "k", k); err != nil {
			return err
		}
		if err := validateExpK("/exp", k); err != nil {
			return err
		}
		if err := validateFinite("/exp", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/exp", ":end", end); err != nil {
			return err
		}
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
		if err := validateNF("/log", nf); err != nil {
			return err
		}
		if err := validateFinite("/log", "k", k); err != nil {
			return err
		}
		if err := validateFinite("/log", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/log", ":end", end); err != nil {
			return err
		}
		if k <= -1 {
			return fmt.Errorf("/log: k must be > -1 to keep (1+k*x) positive (got %v)", k)
		}
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
		if err := validateNF("/cos", nf); err != nil {
			return err
		}
		if err := validateFinite("/cos", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/cos", ":end", end); err != nil {
			return err
		}
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
		if err := validateNF("/pow", nf); err != nil {
			return err
		}
		if err := validateFinite("/pow", "p", p); err != nil {
			return err
		}
		if err := validatePowP("/pow", p); err != nil {
			return err
		}
		if err := validateFinite("/pow", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/pow", ":end", end); err != nil {
			return err
		}
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
		if err := validateNF("/sigmoid", nf); err != nil {
			return err
		}
		if err := validateFinite("/sigmoid", "k", k); err != nil {
			return err
		}
		if err := validateSigmoidK("/sigmoid", k); err != nil {
			return err
		}
		if err := validateFinite("/sigmoid", ":start", start); err != nil {
			return err
		}
		if err := validateFinite("/sigmoid", ":end", end); err != nil {
			return err
		}
		vm.Push(envseg(start, end, nf, func(x float64) float64 {
			return 1 / (1 + math.Exp(-k*(x-0.5)))
		}))
		return nil
	})
}
