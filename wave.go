package main

import (
	"fmt"
	"math"

	"github.com/mjibson/go-dsp/fft"
)

// DefaultWaveSize defines the size of builtin waves
const DefaultWaveSize = 8192

// Wave is a single-cycle waveform.
//
// Typically several such waves are kept at each mip level of a
// wavetable and the oscillator morphs between them as it is
// generating sound.
type Wave []Smp

func (wave Wave) getVal() Val { return wave }

func (wave Wave) String() string {
	return fmt.Sprintf("Wave(size=%d)", len(wave))
}

// removeDCInPlace subtracts the mean from the wave to center it at 0.
func (wave Wave) removeDCInPlace() {
	n := len(wave)
	if n == 0 {
		return
	}
	sum := 0.0
	for _, v := range wave {
		sum += float64(v)
	}
	mean := sum / float64(n)
	if math.Abs(mean) < 1e-12 {
		return
	}
	for i := range wave {
		wave[i] -= Smp(mean)
	}
}

// sample returns a sample from a single wave at fractional phase [0,1).
// Uses 4-point Lagrange (Catmull-Rom) interpolation when possible; falls back to linear for very short waves.
func (wave Wave) sampleAt(phase Smp) Smp {
	n := len(wave)
	if n == 0 {
		return 0
	}
	p := math.Mod(float64(phase), 1.0)
	if p < 0 {
		p += 1.0
	}
	pos := p * float64(n)
	i0 := int(pos) % n
	frac := pos - float64(i0)

	// For tiny waves, just do linear.
	if n < 4 {
		i1 := (i0 + 1) % n
		return wave[i0]*(1.0-frac) + wave[i1]*frac
	}

	// 4-point Catmull-Rom (equivalent to cubic Lagrange with uniform parameterization).
	im1 := (i0 - 1 + n) % n
	i1 := (i0 + 1) % n
	i2 := (i0 + 2) % n
	t := frac
	a0 := -0.5*wave[im1] + 1.5*wave[i0] - 1.5*wave[i1] + 0.5*wave[i2]
	a1 := wave[im1] - 2.5*wave[i0] + 2.0*wave[i1] - 0.5*wave[i2]
	a2 := -0.5*wave[im1] + 0.5*wave[i1]
	a3 := wave[i0]
	return ((a0*t+a1)*t+a2)*t + a3
}

// buildFFTLowpass takes a wave and returns a half-size, lowpassed version using FFT bin masking.
// It zeros bins above half the Nyquist of the previous level and downsamples by 2.
func (wave Wave) buildFFTLowpass() Wave {
	n := len(wave)
	if n <= 1 {
		return append(Wave(nil), wave...)
	}
	// FFT expects complex input.
	x := make([]complex128, n)
	for i, v := range wave {
		x[i] = complex(float64(v), 0)
	}
	X := fft.FFT(x)

	// Zero upper half of bins (simple brickwall at N/4 of original sample rate, since we will downsample by 2).
	for k := n/4 + 1; k < n-(n/4); k++ {
		X[k] = 0
	}

	// IFFT back.
	xt := fft.IFFT(X)
	// Downsample by 2 with implicit box filter from lowpass.
	nextN := n / 2
	out := make(Wave, nextN)
	for i := range nextN {
		// fft.IFFT divides by N; xt[2*i] has that scaling already.
		out[i] = Smp(real(xt[2*i]))
	}
	out.removeDCInPlace()
	return out
}

func sinWave(size int) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	wave := make(Wave, size)
	for i := range size {
		wave[i] = math.Sin(2 * math.Pi * float64(i) / float64(size))
	}
	return wave
}

func tanhWave(size int) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	wave := sinWave(size)
	for i := range wave {
		wave[i] = math.Tanh(wave[i])
	}
	wave.removeDCInPlace()
	return wave
}

func triangleWave(size int) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	wave := make(Wave, size)
	quarter := size / 4
	for i := range quarter {
		t := float64(i) / float64(quarter)
		wave[i] = 1 - t
		wave[i+quarter] = -t
		wave[i+2*quarter] = t - 1
		wave[i+3*quarter] = t
	}
	wave.removeDCInPlace()
	return wave
}

func squareWave(size int) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	wave := make(Wave, size)
	quarter := size / 4
	for i := range quarter {
		wave[i] = 1
		wave[i+quarter] = -1
		wave[i+2*quarter] = -1
		wave[i+3*quarter] = 1
	}
	wave.removeDCInPlace()
	return wave
}

func pulseWave(size int, pw float64) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	if pw < 0 {
		pw = 0
	}
	if pw > 1 {
		pw = 1
	}
	onSamples := int(math.Round(pw * float64(size)))
	wave := make(Wave, size)
	for i := range size {
		if i < onSamples {
			wave[i] = 1
		} else {
			wave[i] = -1
		}
	}
	wave.removeDCInPlace()
	return wave
}

func sawWave(size int) Wave {
	if size == 0 {
		size = DefaultWaveSize
	}
	wave := make(Wave, size)
	half := size / 2
	for i := range half {
		t := float64(i) / float64(half)
		wave[(i+size/4)%size] = t - 1
		wave[(i+half+size/4)%size] = t
	}
	wave.removeDCInPlace()
	return wave
}
