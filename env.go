package main

func init() {
	RegisterWord("line", func(vm *VM) error {
		end := Smp(Pop[Num](vm))
		nf := int(Pop[Num](vm))
		curr := Smp(Pop[Num](vm))
		incr := (end - curr) / Smp(nf)
		t := pushTape(vm, 1, nf)
		for i := range nf {
			t.samples[i] = curr
			curr += incr
		}
		vm.Push(end)
		return nil
	})
}
