package main

import (
	"fmt"
	"math"
)

type Tape struct {
	nchannels int
	nframes   int
	samples   []Smp
}

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(nchannels=%d nframes=%d)", t.nchannels, t.nframes)
}

func (t *Tape) GetSampleIterator() SampleIterator {
	sampleIndex := 0
	end := t.nframes * t.nchannels
	if t.nchannels == 1 {
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
				for range t.nchannels {
					nextSample += t.samples[sampleIndex]
					sampleIndex++
				}
				return nextSample / float64(t.nchannels)
			} else {
				return 0
			}
		}
	}
}

func clamp(value float64, lo float64, hi float64) float64 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

func calcSin(phase float64) float64 {
	return math.Sin(phase * 2 * math.Pi)
}

func calcPulse(phase, width float64) float64 {
	if phase < width {
		return -1.0
	} else {
		return 1.0
	}
}

func calcTriangle(phase float64) float64 {
	if phase < 0.25 {
		return phase * 4.0
	} else if phase < 0.75 {
		return 1.0 - (phase-0.25)*4.0
	} else {
		return -1.0 + (phase-0.75)*4.0
	}
}

func calcSaw(phase float64) float64 {
	if phase < 0.5 {
		return phase * 2.0
	} else {
		return -1.0 + (phase-0.5)*2.0
	}
}

func (t *Tape) GetMessageHandler(msg string, nargs int) Fun {
	if nargs == 0 {
		switch msg {
		case "pulse":
			return func(vm *VM) error {
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				width := clamp(vm.GetFloat(":width"), 0, 1)
				incr := 1.0 / float64(t.nframes)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcPulse(phase, width)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					phase += incr
				}
				return nil
			}
		case "lfo.pulse":
			return func(vm *VM) error {
				sr := vm.GetFloat(":sr")
				freq := GetSampleIterator(vm.GetVal(":freq"))
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				width := clamp(vm.GetFloat(":width"), 0, 1)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcPulse(phase, width)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					incr := 1.0 / (sr / freq())
					phase = math.Mod(phase+incr, 1.0)
				}
				return nil
			}
		case "triangle":
			return func(vm *VM) error {
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				incr := 1.0 / float64(t.nframes)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcTriangle(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					phase += incr
				}
				return nil
			}
		case "lfo.triangle":
			return func(vm *VM) error {
				sr := vm.GetFloat(":sr")
				freq := GetSampleIterator(vm.GetVal(":freq"))
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcTriangle(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					incr := 1.0 / (sr / freq())
					phase = math.Mod(phase+incr, 1.0)
				}
				return nil
			}
		case "saw":
			return func(vm *VM) error {
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				incr := 1.0 / float64(t.nframes)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcSaw(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					phase += incr
				}
				return nil
			}
		case "lfo.saw":
			return func(vm *VM) error {
				sr := vm.GetFloat(":sr")
				freq := GetSampleIterator(vm.GetVal(":freq"))
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcSaw(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					incr := 1.0 / (sr / freq())
					phase = math.Mod(phase+incr, 1.0)
				}
				return nil
			}
		case "sin":
			return func(vm *VM) error {
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				incr := 1.0 / float64(t.nframes)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcSin(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
					}
					phase += incr
				}
				return nil
			}
		case "lfo.sin":
			return func(vm *VM) error {
				sr := vm.GetFloat(":sr")
				freq := GetSampleIterator(vm.GetVal(":freq"))
				phase := clamp(vm.GetFloat(":phase"), 0, 1)
				writeIndex := 0
				for i := 0; i < t.nframes; {
					smp := calcSin(phase)
					for range t.nchannels {
						t.samples[writeIndex] = smp
						writeIndex++
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

func pushTape(vm *VM, nchannels, nframes int) {
	samples := make([]Smp, nchannels*nframes)
	tape := &Tape{
		nchannels: nchannels,
		nframes:   nframes,
		samples:   samples,
	}
	vm.PushVal(tape)
}

func wordTape1(vm *VM) error {
	nframes := int(vm.PopNum())
	pushTape(vm, 1, nframes)
	return nil
}

func wordTape2(vm *VM) error {
	nframes := int(vm.PopNum())
	pushTape(vm, 2, nframes)
	return nil
}

func init() {
	rootEnv.SetVal("tape1", wordTape1)
	rootEnv.SetVal("tape2", wordTape2)
}
