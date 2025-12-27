package main

import (
	"fmt"
	"math"
)

// computeDetuneRatios builds symmetric detune ratios around 1.0 using a spread in cents.
func computeDetuneRatios(voices int, cents float64) []float64 {
	if voices <= 1 {
		return []float64{1.0}
	}
	offsets := make([]float64, voices)
	spread := cents
	if spread < 0 {
		spread = 0
	}
	if voices == 2 {
		offsets[0] = -spread
		offsets[1] = spread
	} else {
		step := 0.0
		if voices > 1 {
			step = (2 * spread) / float64(voices-1)
		}
		for i := range voices {
			offsets[i] = -spread + float64(i)*step
		}
	}
	ratios := make([]float64, voices)
	for i, c := range offsets {
		ratios[i] = math.Pow(2.0, c/1200.0)
	}
	return ratios
}

// computePans returns pan positions in [-spread, spread].
func computePans(voices int, spread float64) []float64 {
	if voices <= 1 || spread <= 0 {
		return []float64{0}
	}
	if spread > 1 {
		spread = 1
	}
	pans := make([]float64, voices)
	if voices == 2 {
		pans[0] = -spread
		pans[1] = spread
		return pans
	}
	step := 0.0
	if voices > 1 {
		step = 2 * spread / float64(voices-1)
	}
	for i := range voices {
		pans[i] = -spread + float64(i)*step
	}
	return pans
}

// deterministicRand returns a deterministic pseudo-random value in [0,1) from an int seed.
func deterministicRand(seed int) float64 {
	// xorshift32
	x := uint32(seed + 1)
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	return float64(x) / float64(^uint32(0))
}

func init() {
	RegisterWord("unison", func(vm *VM) error {
		body := vm.Pop()
		voiceGen, ok := body.(Evaler)
		if !ok {
			return fmt.Errorf("unison: expected closure on stack, got %T", body)
		}
		voices := 1
		if v := vm.GetVal(":voices"); v != nil {
			if n, ok := v.(Num); ok {
				voices = int(n)
				if voices < 1 {
					voices = 1
				}
			} else {
				return fmt.Errorf("unison: :voices must be number")
			}
		}
		spread := 0.0
		if v := vm.GetVal(":spread"); v != nil {
			if n, ok := v.(Num); ok {
				spread = float64(n)
				if spread < 0 {
					spread = 0
				}
			} else {
				return fmt.Errorf("unison: :spread must be number")
			}
		}
		detuneCents := 0.0
		if v := vm.GetVal(":detune"); v != nil {
			if n, ok := v.(Num); ok {
				detuneCents = float64(n)
			} else {
				return fmt.Errorf("unison: :detune must be number (cents)")
			}
		}

		phaseRand := 0.0
		if v := vm.GetVal(":phaseRand"); v != nil {
			if n, ok := v.(Num); ok {
				phaseRand = float64(n)
				if phaseRand < 0 {
					phaseRand = 0
				}
				if phaseRand > 1 {
					phaseRand = 1
				}
			} else {
				return fmt.Errorf("unison: :phaseRand must be number (0..1)")
			}
		}

		baseFreqVal := vm.GetVal(":freq")
		if baseFreqVal == nil {
			return fmt.Errorf("unison: :freq not set")
		}
		baseFreqStream, err := streamFromVal(baseFreqVal)
		if err != nil {
			return fmt.Errorf("unison: cannot use :freq: %w", err)
		}

		ratios := computeDetuneRatios(voices, detuneCents)
		pans := computePans(voices, spread)
		panLR := make([][2]float64, voices)
		for i := 0; i < voices; i++ {
			l, r := equalPowerPan(pans[i])
			panLR[i][0] = l
			panLR[i][1] = r
		}

		voiceStreams := make([]Stream, 0, voices)
		for i := 0; i < voices; i++ {
			if err := vm.DoPushEnv(); err != nil {
				return err
			}
			// Set per-voice detuned freq
			if baseNum, ok := baseFreqVal.(Num); ok {
				vm.SetVal(":freq", Num(float64(baseNum)*ratios[i]))
			} else {
				scaled := baseFreqStream.Combine(Num(ratios[i]).Stream(), MulOp())
				vm.SetVal(":freq", scaled)
			}
			if err := voiceGen.Eval(vm); err != nil {
				vm.DoPopEnv()
				return err
			}
			voiceVal := vm.Pop()
			vm.DoPopEnv()
			vs, err := streamFromVal(voiceVal)
			if err != nil {
				return fmt.Errorf("unison: voice %d did not yield a stream: %w", i, err)
			}
			voiceStream := vs.WithNChannels(1)
			// Apply deterministic random initial phase via delay when base freq is numeric.
			if phaseRand > 0 {
				if baseFreqNum, ok := baseFreqVal.(Num); ok {
					freq := float64(baseFreqNum) * ratios[i]
					if freq > 0 {
						periodSamples := float64(SampleRate()) / freq
						phaseOffset := deterministicRand(i) * phaseRand // fraction of cycle
						delayFrames := int(phaseOffset * periodSamples)
						if delayFrames > 0 {
							voiceStream = voiceStream.Delay(delayFrames)
						}
					}
				}
			}
			voiceStreams = append(voiceStreams, voiceStream)
		}

		// Mix voices into stereo
		nexts := make([]Stepper, len(voiceStreams))
		for i, vs := range voiceStreams {
			nexts[i] = vs.Mono().Next
		}
		mix := makeStream(2, 0, func() (Frame, bool) {
			out := make(Frame, 2)
			norm := 1.0 / float64(len(voiceStreams))
			var lsum, rsum Smp
			for i := range voiceStreams {
				frame, ok := nexts[i]()
				if !ok {
					return nil, false
				}
				s := frame[0]
				lsum += s * panLR[i][0]
				rsum += s * panLR[i][1]
			}
			out[0] = Smp(lsum * norm)
			out[1] = Smp(rsum * norm)
			return out, true
		})

		vm.Push(mix)
		return nil
	})
}
