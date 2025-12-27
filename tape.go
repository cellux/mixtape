package main

import (
	"encoding/binary"
	"fmt"
	"github.com/dh1tw/gosamplerate"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	gl "github.com/go-gl/gl/v3.1/gles2"
	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/go-mp3"
	"github.com/mitchellh/go-homedir"
	"github.com/mjibson/go-dsp/fft"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// DefaultWaveSize defines the size of builtin single-cycle waveforms
const DefaultWaveSize = 8192

type Tape struct {
	nchannels int
	nframes   int
	samples   []Smp
}

type TapeProvider interface {
	Val
	Tape() *Tape
}

func (t *Tape) Tape() *Tape { return t }

func makeTape(nchannels, nframes int) *Tape {
	samples := make([]Smp, nchannels*nframes)
	return &Tape{
		nchannels: nchannels,
		nframes:   nframes,
		samples:   samples,
	}
}

func pushTape(vm *VM, nchannels, nframes int) *Tape {
	tape := makeTape(nchannels, nframes)
	vm.Push(tape)
	return tape
}

func (t *Tape) getVal() Val { return t }

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(nchannels=%d nframes=%d)", t.nchannels, t.nframes)
}

func (t *Tape) Stream() Stream {
	nc := t.nchannels
	nf := t.nframes
	return makeRewindableStream(nc, nf, func() Stepper {
		index := 0
		return func() (Frame, bool) {
			if index >= nf*nc {
				return nil, false
			}
			frame := t.samples[index : index+nc]
			index += nc
			return frame, true
		}
	})
}

// removeDCInPlace subtracts the mean from each channel of the tape to center channels at 0.
func (t *Tape) removeDCInPlace() {
	nf := t.nframes
	if nf == 0 {
		return
	}
	nc := t.nchannels
	smps := t.samples
	sum := make(Frame, nc)
	readIndex := 0
	for range nf {
		for ch := range nc {
			sum[ch] += smps[readIndex]
			readIndex++
		}
	}
	for ch := range nc {
		mean := sum[ch] / Smp(nf)
		if math.Abs(float64(mean)) < 1e-12 {
			continue
		}
		writeIndex := ch
		for range nf {
			smps[writeIndex] -= mean
			writeIndex += nc
		}
	}
}

// GetInterpolatedFrameAtIndex writes the frame at the given fractional `index` to `out`.
// Uses 4-point Lagrange (Catmull-Rom) interpolation when possible; falls back to linear for very short tapes.
// Writes zeroes to `out` if index is out of range or the number of channels in `out` does not match the tape.
func (t *Tape) GetInterpolatedFrameAtIndex(index float64, out Frame) {
	nc := t.nchannels
	nf := t.nframes

	if len(out) != nc || index < 0 || index >= float64(nf) {
		for ch := range out {
			out[ch] = 0
		}
		return
	}

	i0 := int(index) % nf
	frac := Smp(index) - Smp(i0)

	smps := t.samples

	// For tiny waves, just do linear.
	if nf < 4 {
		i1 := i0 + 1
		if i1 == nf {
			i1 = 0
		}
		base0 := i0 * nc
		base1 := i1 * nc
		for ch := range nc {
			out[ch] = smps[base0+ch]*(1.0-frac) + smps[base1+ch]*frac
		}
		return
	}

	// 4-point Catmull-Rom (equivalent to cubic Lagrange with uniform parameterization).
	im1 := i0 - 1
	if im1 < 0 {
		im1 += nf
	}
	i1 := i0 + 1
	if i1 == nf {
		i1 = 0
	}
	i2 := i1 + 1
	if i2 == nf {
		i2 = 0
	}

	baseM1 := im1 * nc
	base0 := i0 * nc
	base1 := i1 * nc
	base2 := i2 * nc
	f := frac
	for ch := range nc {
		a0 := -0.5*smps[baseM1+ch] + 1.5*smps[base0+ch] - 1.5*smps[base1+ch] + 0.5*smps[base2+ch]
		a1 := smps[baseM1+ch] - 2.5*smps[base0+ch] + 2.0*smps[base1+ch] - 0.5*smps[base2+ch]
		a2 := -0.5*smps[baseM1+ch] + 0.5*smps[base1+ch]
		a3 := smps[base0+ch]
		out[ch] = ((a0*f+a1)*f+a2)*f + a3
	}
}

// GetInterpolatedFrameAtPhase writes the frame at the given fractional `phase` to `out`.
// Uses 4-point Lagrange (Catmull-Rom) interpolation when possible; falls back to linear for very short tapes.
// Phase should be in the range [0,1). Writes zeroes to `out` if phase is out of range or the number of channels in out does not match the tape.
func (t *Tape) GetInterpolatedFrameAtPhase(phase float64, out Frame) {
	index := phase * float64(t.nframes)
	t.GetInterpolatedFrameAtIndex(index, out)
}

// AtPhase returns a stream of frames interpolated at fractional phase [0,1).
func (t *Tape) AtPhase(phase Stream) Stream {
	nc := t.nchannels
	nf := t.nframes
	if nf == 0 {
		return makeEmptyStream(nc)
	}
	return makeTransformStream([]Stream{phase}, func(inputs []Stream) Stepper {
		pnext := inputs[0].Next
		out := make(Frame, nc)
		return func() (Frame, bool) {
			frame, ok := pnext()
			if !ok {
				return nil, false
			}
			p := math.Mod(float64(frame[0]), 1.0)
			if p < 0 {
				p += 1.0
			}
			t.GetInterpolatedFrameAtPhase(p, out)
			return out, true
		}
	})
}

// buildFFTLowpass takes a single-channel tape and returns a
// half-size, lowpassed version using FFT bin masking.  It zeroess
// bins above half the Nyquist and downsamples by 2.
// if t is not single-channel or its frame count <=1, returns t.
func (t *Tape) buildFFTLowpass() *Tape {
	if t.nchannels != 1 || t.nframes <= 1 {
		return t
	}
	nf := t.nframes
	// FFT expects complex input.
	x := make([]complex128, nf)
	for i, v := range t.samples {
		x[i] = complex(Num(v), 0)
	}
	X := fft.FFT(x)

	// Zero upper half of bins (simple brickwall at N/4 of original
	// sample rate, since we will downsample by 2).
	for k := nf/4 + 1; k < nf-(nf/4); k++ {
		X[k] = 0
	}

	// IFFT back.
	xt := fft.IFFT(X)
	// Downsample by 2 with implicit box filter from lowpass.
	nextN := nf / 2
	out := makeTape(1, nextN)
	for i := range nextN {
		// fft.IFFT divides by N; xt[2*i] has that scaling already.
		out.samples[i] = Smp(real(xt[2*i]))
	}
	out.removeDCInPlace()
	return out
}

func sinTape(size int) *Tape {
	if size == 0 {
		size = DefaultWaveSize
	}
	t := makeTape(1, size)
	for i := range size {
		t.samples[i] = math.Sin(2 * math.Pi * float64(i) / float64(size))
	}
	return t
}

func tanhTape(size int) *Tape {
	if size == 0 {
		size = DefaultWaveSize
	}
	t := sinTape(size)
	for i := range t.nframes {
		t.samples[i] = math.Tanh(t.samples[i])
	}
	return t
}

func triangleTape(size int) *Tape {
	if size == 0 {
		size = DefaultWaveSize
	}
	t := makeTape(1, size)
	quarter := size / 4
	for i := range quarter {
		t0 := Smp(i) / Smp(quarter)
		t.samples[i+0*quarter] = t0
		t.samples[i+1*quarter] = 1 - t0
		t.samples[i+2*quarter] = -t0
		t.samples[i+3*quarter] = t0 - 1
	}
	return t
}

func squareTape(size int) *Tape {
	if size == 0 {
		size = DefaultWaveSize
	}
	t := makeTape(1, size)
	quarter := size / 4
	for i := range quarter {
		t.samples[i] = 1
		t.samples[i+quarter] = -1
		t.samples[i+2*quarter] = -1
		t.samples[i+3*quarter] = 1
	}
	return t
}

func pulseTape(size int, pw float64) *Tape {
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
	t := makeTape(1, size)
	for i := range size {
		if i < onSamples {
			t.samples[i] = 1
		} else {
			t.samples[i] = -1
		}
	}
	return t
}

func sawTape(size int) *Tape {
	if size == 0 {
		size = DefaultWaveSize
	}
	t := makeTape(1, size)
	half := size / 2
	for i := range half {
		t0 := Smp(i) / Smp(half)
		t.samples[i%size] = t0
		t.samples[(i+half)%size] = t0 - 1
	}
	return t
}

func (t *Tape) Slice(start, end int) *Tape {
	nframes := end - start
	slicedTape := &Tape{
		nchannels: t.nchannels,
		nframes:   nframes,
		samples:   t.samples[start*t.nchannels : end*t.nchannels],
	}
	return slicedTape
}

func (t *Tape) WriteToWav(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sr := SampleRate()
	enc := wav.NewEncoder(f, sr, 16, t.nchannels, 1)
	defer enc.Close()
	nsamples := t.nframes * t.nchannels
	intBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: t.nchannels,
			SampleRate:  sr,
		},
		Data:           make([]int, nsamples),
		SourceBitDepth: 16,
	}
	for i := range nsamples {
		intBuf.Data[i] = int(t.samples[i] * 32767)
	}
	err = enc.Write(intBuf)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	RegisterMethod[*Tape]("shift", 2, func(vm *VM) error {
		amount, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		t, err := Top[*Tape](vm)
		if err != nil {
			return err
		}
		if amount < 0 {
			if amount > -1.0 {
				amount = 1.0 + amount
			} else {
				amount = Num(t.nframes) + amount
			}
		}
		if amount < 1.0 {
			amount = Num(t.nframes) * amount
		}
		amountSamples := int(math.Round(float64(amount))) % t.nframes
		t.samples = append(t.samples[amountSamples:], t.samples[:amountSamples]...)
		return nil
	})

	RegisterNum("SRC_SINC_BEST_QUALITY", 0)
	RegisterNum("SRC_SINC_MEDIUM_QUALITY", 1)
	RegisterNum("SRC_SINC_FASTEST", 2)
	RegisterNum("SRC_ZERO_ORDER_HOLD", 3)
	RegisterNum("SRC_LINEAR", 4)

	RegisterMethod[*Tape]("resample", 3, func(vm *VM) error {
		ratioNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		ratio := float64(ratioNum)
		if ratio <= 0 {
			return fmt.Errorf("resample: invalid ratio: %f", ratio)
		}
		converterTypeNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		converterType := int(converterTypeNum)
		if converterType < 0 || converterType > 4 {
			return fmt.Errorf("resample: invalid converterType: %d - must be between 0..4", converterType)
		}
		t, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		tempBuf := make([]float32, t.nframes*t.nchannels)
		for i, smp := range t.samples {
			tempBuf[i] = float32(smp)
		}
		resampledBuf, err := gosamplerate.Simple(tempBuf, ratio, t.nchannels, converterType)
		if err != nil {
			return err
		}
		resampledFrames := len(resampledBuf) / t.nchannels
		resampledTape := pushTape(vm, t.nchannels, resampledFrames)
		for i, smp := range resampledBuf {
			resampledTape.samples[i] = Smp(smp)
		}
		return nil
	})
}

func expandPath(path string) (string, error) {
	p, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	return os.ExpandEnv(p), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveTapePath(path string) (string, error) {
	p, err := expandPath(path)
	if err != nil {
		return "", err
	}
	for _, ext := range []string{".tape", ".wav", ".mp3"} {
		if filepath.Ext(p) == ext {
			return p, nil
		}
		pathWithExt := fmt.Sprintf("%s%s", p, ext)
		if fileExists(pathWithExt) {
			return pathWithExt, nil
		}
	}
	return "", fmt.Errorf("tape not found: %s", path)
}

func loadTape(vm *VM, path string) error {
	tapeInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	wavPath := fmt.Sprintf("%s.wav", strings.TrimSuffix(path, ".tape"))
	if wavInfo, err := os.Stat(wavPath); err == nil {
		if wavInfo.ModTime().After(tapeInfo.ModTime()) {
			return loadAndPushTape(vm, wavPath)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := vm.ParseAndEval(f, path); err != nil {
		return err
	}
	if tape, ok := vm.Top().(*Tape); ok {
		if err := tape.WriteToWav(wavPath); err != nil {
			return err
		}
	}
	return nil
}

func loadWav(vm *VM, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sr := SampleRate()
	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return fmt.Errorf("invalid WAV file: %s", path)
	}
	if err := decoder.FwdToPCM(); err != nil {
		return err
	}
	format := decoder.Format()
	bitDepth := int(decoder.SampleBitDepth())
	if bitDepth == 0 {
		return fmt.Errorf("unknown bit depth for WAV file: %s", path)
	}
	bytesPerSample := (bitDepth-1)/8 + 1
	nbytes := int(decoder.PCMLen())
	nsamples := nbytes / bytesPerSample
	nchannels := format.NumChannels
	nframes := nsamples / nchannels
	logger.Debug("decoding wav file",
		"path", path,
		"sampleRate", format.SampleRate,
		"nchannels", format.NumChannels,
		"bitDepth", bitDepth,
		"bytesPerSample", bytesPerSample,
		"nbytes", nbytes,
		"nsamples", nsamples,
		"nframes", nframes,
	)
	startTime := GetTime()
	buf := &audio.IntBuffer{
		Format:         format,
		Data:           make([]int, nsamples),
		SourceBitDepth: 16,
	}
	bytesDecoded, err := decoder.PCMBuffer(buf)
	if err != nil {
		return err
	}
	logger.Debug("decoded wav file", "path", path, "seconds", GetTime()-startTime, "bytesDecoded", bytesDecoded)
	floatBuf := buf.AsFloatBuffer()
	factor := math.Pow(2, float64(bitDepth-1))
	wavSR := buf.Format.SampleRate
	if wavSR != sr {
		float32Buf := make([]float32, nchannels*nframes)
		for i := 0; i < len(floatBuf.Data); i++ {
			float32Buf[i] = float32(floatBuf.Data[i] / factor)
		}
		logger.Debug("resampling wav data", "path", path)
		startTime = GetTime()
		resampledBuf, err := gosamplerate.Simple(float32Buf, float64(sr)/float64(wavSR), nchannels, gosamplerate.SRC_SINC_BEST_QUALITY)
		if err != nil {
			return err
		}
		logger.Debug("resampled wav data", "path", path, "seconds", GetTime()-startTime)
		nsamples := len(resampledBuf)
		nframes := nsamples / nchannels
		tape := pushTape(vm, nchannels, nframes)
		for i := range nsamples {
			tape.samples[i] = Smp(resampledBuf[i])
		}
		return nil
	}

	tape := pushTape(vm, nchannels, nframes)
	for i := 0; i < len(floatBuf.Data); i++ {
		tape.samples[i] = Smp(floatBuf.Data[i] / factor)
	}
	return nil
}

func loadMP3(vm *VM, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sr := SampleRate()
	decoder, err := mp3.NewDecoder(f)
	if err != nil {
		return err
	}
	nbytes := decoder.Length()
	if nbytes <= 0 {
		return fmt.Errorf("cannot determine length of MP3 file: %s", path)
	}
	nchannels := 2
	nsamples := int(nbytes / 2) // FormatSignedInt16LE
	nframes := nsamples / nchannels
	mp3SR := decoder.SampleRate()
	if mp3SR != sr {
		logger.Debug("decoding mp3 file", "path", path)
		startTime := GetTime()
		float32Buf := make([]float32, nsamples)
		var sample int16
		for i := range nsamples {
			if err := binary.Read(decoder, binary.LittleEndian, &sample); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			float32Buf[i] = float32(sample) / 32768
		}
		logger.Debug("decoded mp3 file", "path", path, "seconds", GetTime()-startTime)
		startTime = GetTime()
		logger.Debug("resampling mp3 data", "path", path)
		resampledBuf, err := gosamplerate.Simple(float32Buf, float64(sr)/float64(mp3SR), nchannels, gosamplerate.SRC_SINC_BEST_QUALITY)
		if err != nil {
			return err
		}
		logger.Debug("resampled mp3 data", "path", path, "seconds", GetTime()-startTime)
		nsamples := len(resampledBuf)
		nframes := nsamples / nchannels
		tape := pushTape(vm, nchannels, nframes)
		for i := range nsamples {
			tape.samples[i] = Smp(resampledBuf[i])
		}
		return nil
	}

	logger.Debug("decoding mp3 file", "path", path)
	startTime := GetTime()
	var sample int16
	tape := pushTape(vm, nchannels, nframes)
	for i := range nsamples {
		if err := binary.Read(decoder, binary.LittleEndian, &sample); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		tape.samples[i] = Smp(sample) / 32768
	}
	logger.Debug("decoded mp3 file", "path", path, "seconds", GetTime()-startTime)
	return nil
}

func loadAndPushTape(vm *VM, path string) error {
	switch filepath.Ext(path) {
	case ".tape":
		return loadTape(vm, path)
	case ".wav":
		return loadWav(vm, path)
	case ".mp3":
		return loadMP3(vm, path)
	default:
		return fmt.Errorf("cannot load file: %s", path)
	}
}

func init() {
	RegisterMethod[Str]("load", 1, func(vm *VM) error {
		pathVal, err := Pop[Str](vm)
		if err != nil {
			return err
		}
		path, err := resolveTapePath(string(pathVal))
		if err != nil {
			return err
		}
		return loadAndPushTape(vm, path)
	})
}

type TapeReader struct {
	tape          *Tape
	tapeOffset    int
	audioChannels int
	audioOffset   int
}

func writeSampleAsFloat32bits(buf []byte, index int, smp Smp) {
	u32smp := math.Float32bits(float32(smp))
	buf[index] = byte(u32smp)
	buf[index+1] = byte(u32smp >> 8)
	buf[index+2] = byte(u32smp >> 16)
	buf[index+3] = byte(u32smp >> 24)
}

func (tr *TapeReader) GetCurrentFrame(bytesStillInAudioBuffer int) int {
	samplesStillInAudioBuffer := bytesStillInAudioBuffer / 4
	return (tr.audioOffset - samplesStillInAudioBuffer) / tr.audioChannels
}

func (tr *TapeReader) Read(buf []byte) (int, error) {
	samples := tr.tape.samples
	tapeOffset := tr.tapeOffset
	audioOffset := tr.audioOffset
	samplesLeft := len(samples) - tapeOffset
	if samplesLeft == 0 {
		logger.Debug("playing finished")
		return 0, io.EOF
	}
	bufLengthInSamples := len(buf) / 4
	writeIndex := 0
	srcChannels := tr.tape.nchannels
	dstChannels := tr.audioChannels
	switch srcChannels {
	case 1:
		switch dstChannels {
		case 1:
			framesToWrite := min(bufLengthInSamples, samplesLeft)
			bytesToWrite := framesToWrite * 4
			for writeIndex < bytesToWrite {
				smp := samples[tapeOffset]
				tapeOffset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
			}
		case 2:
			framesToWrite := min(bufLengthInSamples/2, samplesLeft)
			bytesToWrite := framesToWrite * 8
			for writeIndex < bytesToWrite {
				smp := samples[tapeOffset]
				tapeOffset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
			}
		}
	case 2:
		switch dstChannels {
		case 1:
			framesToWrite := min(bufLengthInSamples, samplesLeft/2)
			bytesToWrite := framesToWrite * 4
			for writeIndex < bytesToWrite {
				smp := (samples[tapeOffset] + samples[tapeOffset+1]) / 2.0
				tapeOffset += 2
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
			}
		case 2:
			framesToWrite := min(bufLengthInSamples/2, samplesLeft/2)
			bytesToWrite := framesToWrite * 8
			for writeIndex < bytesToWrite {
				smp := samples[tapeOffset]
				tapeOffset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
				smp = samples[tapeOffset]
				tapeOffset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				audioOffset++
			}
		}
	}
	tr.tapeOffset = tapeOffset
	tr.audioOffset = audioOffset
	return writeIndex, nil
}

func MakeTapeReader(tape *Tape, nchannels int) *TapeReader {
	return &TapeReader{
		tape:          tape,
		tapeOffset:    0,
		audioChannels: nchannels,
		audioOffset:   0,
	}
}

func init() {
	RegisterMethod[TapeProvider]("tape", 1, func(vm *VM) error {
		tp, err := Pop[TapeProvider](vm)
		if err != nil {
			return err
		}
		vm.Push(tp.Tape())
		return nil
	})

	RegisterWord("tape1", func(vm *VM) error {
		nframesNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		pushTape(vm, 1, int(nframesNum))
		return nil
	})

	RegisterWord("tape2", func(vm *VM) error {
		nframesNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		pushTape(vm, 2, int(nframesNum))
		return nil
	})

	RegisterWord("tape/sin", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		vm.Push(sinTape(int(size)))
		return nil
	})

	RegisterWord("tape/tanh", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		vm.Push(tanhTape(int(size)))
		return nil
	})

	RegisterWord("tape/triangle", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		vm.Push(triangleTape(int(size)))
		return nil
	})

	RegisterWord("tape/square", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		vm.Push(squareTape(int(size)))
		return nil
	})

	RegisterWord("tape/pulse", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		pw := 0.5
		if pwVal := vm.GetVal(":pw"); pwVal != nil {
			if pwNum, ok := pwVal.(Num); ok {
				pw = float64(pwNum)
			} else {
				return fmt.Errorf("tape/pulse: :pw must be number")
			}
		}
		vm.Push(pulseTape(int(size), pw))
		return nil
	})

	RegisterWord("tape/saw", func(vm *VM) error {
		size, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		vm.Push(sawTape(int(size)))
		return nil
	})

	RegisterMethod[*Tape]("at", 2, func(vm *VM) error {
		indexNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		t, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		index := int(indexNum)
		if index < 0 || index >= t.nframes {
			return vm.Errorf("Tape.at: invalid frame index: %d", index)
		}
		f := make(Frame, t.nchannels)
		t.GetInterpolatedFrameAtIndex(float64(indexNum), f)
		out := make(Vec, t.nchannels)
		for ch := range t.nchannels {
			out[ch] = Num(f[ch])
		}
		vm.Push(out)
		return nil
	})

	RegisterMethod[*Tape]("at/phase", 2, func(vm *VM) error {
		phase, err := streamFromVal(vm.Pop())
		if err != nil {
			return err
		}
		t, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		vm.Push(t.AtPhase(phase))
		return nil
	})

	RegisterMethod[*Tape]("slice", 3, func(vm *VM) error {
		endNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		startNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		t, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		vm.Push(t.Slice(int(startNum), int(endNum)))
		return nil
	})

	RegisterMethod[*Tape]("+@", 3, func(vm *VM) error {
		offsetNum, err := Pop[Num](vm)
		if err != nil {
			return err
		}
		rhs, err := Pop[*Tape](vm)
		if err != nil {
			return err
		}
		lhs, err := Top[*Tape](vm)
		if err != nil {
			return err
		}
		offset := int(offsetNum)
		nchannels := lhs.nchannels
		end := offset + rhs.nframes
		if lhs.nframes < end {
			extraFramesNeeded := end - lhs.nframes
			lhs.samples = append(lhs.samples, make([]Smp, extraFramesNeeded*nchannels)...)
			lhs.nframes += extraFramesNeeded
		}
		s := rhs.Stream().WithNChannels(nchannels)
		writeIndex := offset * nchannels
		for frame := range s.Seq() {
			for i := range nchannels {
				lhs.samples[writeIndex] += frame[i]
				writeIndex++
			}
		}
		return nil
	})
}

// TapeDisplay

const (
	pointVertexShader = `
		precision highp float;
		attribute vec2 a_position;
		uniform mat4 u_transform;
		void main(void) {
			gl_Position = u_transform * vec4(a_position, 0.0, 1.0);
		};` + "\x00"
	pointFragmentShader = `
		precision highp float;
		uniform vec4 u_color;
		void main(void) {
			gl_FragColor = u_color;
		};` + "\x00"
)

type PointVertex struct {
	position [2]float32
}

type TapeDisplay struct {
	tape        *Tape
	pixelRect   Rect
	vertices    [][]PointVertex
	program     Program
	a_position  int32
	u_transform int32
	u_color     int32
}

func CreateTapeDisplay() (*TapeDisplay, error) {
	program, err := CreateProgram(pointVertexShader, pointFragmentShader)
	if err != nil {
		return nil, err
	}
	td := &TapeDisplay{
		program:     program,
		a_position:  program.GetAttribLocation("a_position\x00"),
		u_transform: program.GetUniformLocation("u_transform\x00"),
		u_color:     program.GetUniformLocation("u_color\x00"),
	}
	return td, nil
}

func (td *TapeDisplay) Render(tape *Tape, pixelRect Rect, windowSize int, windowOffset int, playheadFrames []int) {
	pixelWidth, pixelHeight := pixelRect.Dx(), pixelRect.Dy()
	if pixelWidth == 0 || pixelHeight == 0 {
		return
	}
	if td.tape != tape || td.pixelRect != pixelRect {
		td.tape = tape
		td.pixelRect = pixelRect
		td.vertices = make([][]PointVertex, tape.nchannels)
		for ch := range tape.nchannels {
			td.vertices[ch] = make([]PointVertex, pixelWidth*2)
			for x := range pixelWidth {
				px := float32(x) + 0.5
				idx := x * 2
				td.vertices[ch][idx].position[0] = px
				td.vertices[ch][idx+1].position[0] = px
			}
		}
	}
	channelHeight := float32(pixelHeight) / float32(tape.nchannels)
	channelHeightHalf := channelHeight / 2.0
	incr := float64(windowSize) / float64(pixelWidth)
	readIndex := float64(windowOffset)
	channelClipped := make([]bool, tape.nchannels)
	for x := range pixelWidth {
		i0 := int(math.Floor(readIndex))
		i1 := int(math.Ceil(readIndex + incr))
		if i1 <= i0 {
			i1 = i0 + 1
		}
		if i0 < 0 {
			i0 = 0
		}
		if i1 > tape.nframes {
			i1 = tape.nframes
		}
		channelTop := float32(0)
		for ch := range tape.nchannels {
			minVal := math.Inf(1)
			maxVal := math.Inf(-1)
			base := ch
			for i := i0; i < i1; i++ {
				smp := float64(tape.samples[base+i*tape.nchannels])
				if smp < minVal {
					minVal = smp
				}
				if smp > maxVal {
					maxVal = smp
				}
			}
			if math.Abs(minVal) > 1.0 || math.Abs(maxVal) > 1.0 {
				channelClipped[ch] = true
			}
			yMin := channelTop + channelHeightHalf - float32(minVal)*channelHeightHalf
			yMax := channelTop + channelHeightHalf - float32(maxVal)*channelHeightHalf

			// When the signal is constant (min == max), our per-column vertical line
			// collapses to a point. gles2 doesn't reliably rasterize zero-length lines,
			// so we expand it to at least ~1 pixel so constant tapes are visible.
			height := yMin - yMax
			if height < 1.0 {
				center := (yMin + yMax) * 0.5
				half := float32(0.5)
				yMin = center + half
				yMax = center - half

				// Clamp to the channel bounds by shifting the segment while
				// preserving its minimum height.
				upper := channelTop + channelHeight
				if yMin > upper {
					shift := yMin - upper
					yMin -= shift
					yMax -= shift
				}
				if yMax < channelTop {
					shift := channelTop - yMax
					yMin += shift
					yMax += shift
				}
			}

			idx := x * 2
			td.vertices[ch][idx].position[1] = yMin
			td.vertices[ch][idx+1].position[1] = yMax
			channelTop += channelHeight
		}
		readIndex += incr
	}
	// Build transform once (pixel space -> clip space)
	ux := 2.0 / float32(fbSize.X)
	uy := 2.0 / float32(fbSize.Y)
	mScale := mgl.Scale3D(ux, -uy, 1)
	tx := -1.0 + ux*float32(pixelRect.Min.X)
	ty := 1.0 - uy*float32(pixelRect.Min.Y)
	mTranslate := mgl.Translate3D(tx, ty, 0)
	mTransform := mTranslate.Mul4(mScale)

	td.program.Use()
	gl.UniformMatrix4fv(td.u_transform, 1, false, &mTransform[0])
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.EnableVertexAttribArray(uint32(td.a_position))

	stride := int32(unsafe.Sizeof(PointVertex{}))

	// Draw faint waveform fill + stroke per channel for a more polished look.
	for ch := range tape.nchannels {
		ptr := gl.Ptr(&td.vertices[ch][0].position[0])
		count := int32(len(td.vertices[ch]))

		// subtle fill
		gl.LineWidth(3.0)
		gl.Uniform4f(td.u_color, 1.0, 1.0, 1.0, 0.16)
		gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, ptr)
		gl.DrawArrays(gl.LINES, 0, count)

		// crisp stroke
		gl.LineWidth(1.0)
		gl.Uniform4f(td.u_color, 1.0, 1.0, 1.0, 0.9)
		gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, ptr)
		gl.DrawArrays(gl.LINES, 0, count)
	}

	// Zero lines and bounds per channel
	lineVerts := [2]PointVertex{{position: [2]float32{0, 0}}, {position: [2]float32{float32(pixelWidth), 0}}}
	for ch := range tape.nchannels {
		channelTop := float32(ch) * channelHeight
		// zero line
		lineVerts[0].position[1] = channelTop + channelHeightHalf
		lineVerts[1].position[1] = channelTop + channelHeightHalf
		gl.Uniform4f(td.u_color, 1.0, 1.0, 1.0, 0.15)
		gl.LineWidth(1.0)
		gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, gl.Ptr(&lineVerts[0].position[0]))
		gl.DrawArrays(gl.LINES, 0, 2)

		// guard lines
		guardColor := [4]float32{1.0, 1.0, 1.0, 0.12}
		if channelClipped[ch] {
			guardColor = [4]float32{1.0, 0.2, 0.2, 0.7}
		}
		gl.Uniform4f(td.u_color, guardColor[0], guardColor[1], guardColor[2], guardColor[3])
		lineVerts[0].position[1] = channelTop
		lineVerts[1].position[1] = channelTop
		gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, gl.Ptr(&lineVerts[0].position[0]))
		gl.DrawArrays(gl.LINES, 0, 2)
		lineVerts[0].position[1] = channelTop + channelHeight
		lineVerts[1].position[1] = channelTop + channelHeight
		gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, gl.Ptr(&lineVerts[0].position[0]))
		gl.DrawArrays(gl.LINES, 0, 2)
	}

	// Playhead indicators
	for _, playheadFrame := range playheadFrames {
		playheadX := int(math.Round(float64(playheadFrame-windowOffset) / incr))
		if playheadX >= 0 && playheadX < pixelWidth {
			px := float32(playheadX) + 0.5
			playheadVerts := [2]PointVertex{{position: [2]float32{px, 0}}, {position: [2]float32{px, float32(gl.SAMPLE_LOCATION_PIXEL_GRID_HEIGHT_NV)}}}
			gl.LineWidth(1.0)
			gl.Uniform4f(td.u_color, 1.0, 1.0, 1.0, 0.5)
			gl.VertexAttribPointer(uint32(td.a_position), 2, gl.FLOAT, false, stride, gl.Ptr(&playheadVerts[0].position[0]))
			gl.DrawArrays(gl.LINES, 0, 2)
		}
	}

	gl.LineWidth(1.0)
	gl.Disable(gl.BLEND)
	gl.DisableVertexAttribArray(uint32(td.a_position))
}
