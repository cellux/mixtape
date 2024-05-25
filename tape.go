package main

import (
	"fmt"
	"math"
)

type Tape struct {
	channels int
	length   int
	samples  []Smp
}

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(channels=%d length=%d)", t.channels, t.length)
}

func (t *Tape) GetSampleIterator() SampleIterator {
	sampleIndex := 0
	end := t.length * t.channels
	if t.channels == 1 {
		return func() Smp {
			if sampleIndex < end {
				nextSample := t.samples[sampleIndex]
				sampleIndex++
				return nextSample
			} else {
				return 0
			}
		}
	} else {
		return func() Smp {
			if sampleIndex < end {
				var nextSample Smp
				for range t.channels {
					nextSample += t.samples[sampleIndex]
					sampleIndex++
				}
				return nextSample / float64(t.channels)
			} else {
				return 0
			}
		}
	}
}

func (t *Tape) GetMessageHandler(msg string, nargs int) Fun {
	if nargs == 0 {
		switch msg {
		case "sin":
			return func(vm *VM) error {
				phase := vm.GetFloat(":phase")
				incr := (2 * math.Pi) / float64(t.length)
				for i := 0; i < t.length; {
					smp := math.Sin(phase)
					for range t.channels {
						t.samples[i] = smp
						i++
					}
					phase += incr
				}
				return nil
			}
		case "osc.sin":
			return func(vm *VM) error {
				sr := vm.GetFloat(":sr")
				freq := GetSampleIterator(vm.GetVal(":freq"))
				phase := vm.GetFloat(":phase")
				for i := 0; i < t.length; {
					smp := math.Sin(phase * 2 * math.Pi)
					for range t.channels {
						t.samples[i] = smp
						i++
					}
					incr := 1.0 / (sr / freq())
					phase = math.Mod(phase+incr, 1.0)
				}
				return nil
			}
		}
	}
	return nil
}

func pushTape(vm *VM, channels, length int) {
	samples := make([]Smp, channels*length)
	tape := &Tape{
		channels: channels,
		length:   length,
		samples:  samples,
	}
	vm.PushVal(tape)
}

func wordTape1(vm *VM) error {
	pushTape(vm, 1, vm.GetInt(":length"))
	return nil
}

func wordTape2(vm *VM) error {
	pushTape(vm, 2, vm.GetInt(":length"))
	return nil
}

func init() {
	rootEnv.SetVal("tape1", wordTape1)
	rootEnv.SetVal("tape2", wordTape2)
}
