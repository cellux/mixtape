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
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"unsafe"
)

type Tape struct {
	sr        float64
	nchannels int
	nframes   int
	samples   []Smp
}

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(sr=%g nchannels=%d nframes=%d)", t.sr, t.nchannels, t.nframes)
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
				return nextSample / Smp(t.nchannels)
			} else {
				return 0
			}
		}
	}
}

func (t *Tape) GetInterpolatedSampleAt(channel int, frame float64) Smp {
	frameIndexLo := math.Floor(frame)
	sampleIndexLo := channel + int(frameIndexLo)*t.nchannels
	smpLo := t.samples[sampleIndexLo]
	smpHi := t.samples[sampleIndexLo+1]
	frameIndexDelta := frame - frameIndexLo
	return smpLo + (smpHi-smpLo)*frameIndexDelta
}

func (t *Tape) WriteToWav(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := wav.NewEncoder(f, int(t.sr), 16, t.nchannels, 1)
	defer enc.Close()
	nsamples := t.nframes * t.nchannels
	intBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: t.nchannels,
			SampleRate:  int(t.sr),
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

func init() {
	RegisterMethod[*Tape]("pulse", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		width := clamp(vm.GetFloat(":width"), 0, 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcPulse(phase, width)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.pulse", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		sr := t.sr
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		width := clamp(vm.GetFloat(":width"), 0, 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcPulse(phase, width)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("triangle", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcTriangle(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.triangle", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		sr := t.sr
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcTriangle(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("saw", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSaw(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.saw", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		sr := t.sr
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSaw(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("sin", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		incr := 1.0 / float64(t.nframes)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSin(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			phase = math.Mod(phase+incr, 1.0)
		}
		return nil
	})

	RegisterMethod[*Tape]("lfo.sin", 1, func(vm *VM) error {
		t := Top[*Tape](vm)
		sr := t.sr
		freq := GetSampleIterator(vm.GetVal(":freq"))
		phase := math.Mod(vm.GetFloat(":phase"), 1)
		writeIndex := 0
		for i := 0; i < t.nframes; i++ {
			smp := calcSin(phase)
			for range t.nchannels {
				t.samples[writeIndex] = smp
				writeIndex++
			}
			incr := 1.0 / (sr / freq())
			phase = math.Mod(phase+incr, 1.0)
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

func loadAndPushTape(vm *VM, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sr := vm.GetFloat(":sr")
	switch filepath.Ext(path) {
	case ".tape":
		tapeInfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		wavPath := fmt.Sprintf("%s.wav", path[:len(path)-5])
		wavInfo, err := os.Stat(wavPath)
		if err == nil {
			if wavInfo.ModTime().After(tapeInfo.ModTime()) {
				return loadAndPushTape(vm, wavPath)
			}
		}
		err = vm.ParseAndExecute(f, path)
		if err != nil {
			return err
		}
		if vm.StackSize() > 0 {
			val := vm.TopVal()
			if tape, ok := val.(*Tape); ok {
				err := tape.WriteToWav(wavPath)
				if err != nil {
					return err
				}
			}
		}
	case ".wav":
		decoder := wav.NewDecoder(f)
		if !decoder.IsValidFile() {
			return fmt.Errorf("invalid WAV file: %s", path)
		}
		err := decoder.FwdToPCM()
		if err != nil {
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
		slog.Info("decoding wav file",
			"path", path,
			"sampleRate", format.SampleRate,
			"nchannels", format.NumChannels,
			"bitDepth", bitDepth,
			"bytesPerSample", bytesPerSample,
			"nbytes", nbytes,
			"nsamples", nsamples,
			"nframes", nframes,
		)
		var startTime float64
		startTime = GetTime()
		buf := &audio.IntBuffer{
			Format:         format,
			Data:           make([]int, nsamples),
			SourceBitDepth: 16,
		}
		bytesDecoded, err := decoder.PCMBuffer(buf)
		if err != nil {
			return err
		}
		slog.Info("decoded wav file", "path", path, "seconds", GetTime()-startTime, "bytesDecoded", bytesDecoded)
		floatBuf := buf.AsFloatBuffer()
		factor := math.Pow(2, float64(bitDepth-1))
		wavSR := float64(buf.Format.SampleRate)
		if wavSR != sr {
			float32Buf := make([]float32, nchannels*nframes)
			for i := 0; i < len(floatBuf.Data); i++ {
				float32Buf[i] = float32(floatBuf.Data[i] / factor)
			}
			slog.Info("resampling wav data", "path", path)
			startTime = GetTime()
			resampledBuf, err := gosamplerate.Simple(float32Buf, sr/wavSR, nchannels, gosamplerate.SRC_SINC_BEST_QUALITY)
			if err != nil {
				return err
			}
			slog.Info("resampled wav data", "path", path, "seconds", GetTime()-startTime)
			nsamples := len(resampledBuf)
			nframes := nsamples / nchannels
			tape := pushTape(vm, sr, nchannels, nframes)
			for i := range nsamples {
				tape.samples[i] = float64(resampledBuf[i])
			}
		} else {
			tape := pushTape(vm, sr, nchannels, nframes)
			for i := 0; i < len(floatBuf.Data); i++ {
				tape.samples[i] = floatBuf.Data[i] / factor
			}
		}
	case ".mp3":
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
		mp3SR := float64(decoder.SampleRate())
		if mp3SR != sr {
			var startTime float64
			slog.Info("decoding mp3 file", "path", path)
			startTime = GetTime()
			float32Buf := make([]float32, nsamples)
			var sample int16
			for i := range nsamples {
				err := binary.Read(decoder, binary.LittleEndian, &sample)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				float32Buf[i] = float32(sample) / 32768
			}
			slog.Info("decoded mp3 file", "path", path, "seconds", GetTime()-startTime)
			startTime = GetTime()
			slog.Info("resampling mp3 data", "path", path)
			resampledBuf, err := gosamplerate.Simple(float32Buf, sr/mp3SR, nchannels, gosamplerate.SRC_SINC_BEST_QUALITY)
			if err != nil {
				return err
			}
			slog.Info("resampled mp3 data", "path", path, "seconds", GetTime()-startTime)
			nsamples := len(resampledBuf)
			nframes := nsamples / nchannels
			tape := pushTape(vm, sr, nchannels, nframes)
			for i := range nsamples {
				tape.samples[i] = float64(resampledBuf[i])
			}
		} else {
			var startTime float64
			slog.Info("decoding mp3 file", "path", path)
			startTime = GetTime()
			var sample int16
			tape := pushTape(vm, sr, nchannels, nframes)
			for i := range nsamples {
				err := binary.Read(decoder, binary.LittleEndian, &sample)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				tape.samples[i] = float64(sample) / 32768
			}
			slog.Info("decoded mp3 file", "path", path, "seconds", GetTime()-startTime)
		}
	default:
		return fmt.Errorf("cannot load file: %s", path)
	}
	return nil
}

func init() {
	RegisterMethod[Str]("load", 1, func(vm *VM) error {
		path, err := resolveTapePath(string(Pop[Str](vm)))
		if err != nil {
			return err
		}
		return loadAndPushTape(vm, path)
	})
}

type TapeReader struct {
	tape      *Tape
	nchannels int
	offset    int
}

func writeSampleAsFloat32bits(buf []byte, index int, smp Smp) {
	u32smp := math.Float32bits(float32(smp))
	buf[index] = byte(u32smp)
	buf[index+1] = byte(u32smp >> 8)
	buf[index+2] = byte(u32smp >> 16)
	buf[index+3] = byte(u32smp >> 24)
}

func (tr *TapeReader) Read(buf []byte) (int, error) {
	samples := tr.tape.samples
	offset := tr.offset
	samplesLeft := len(samples) - offset
	if samplesLeft == 0 {
		slog.Info("playing finished")
		return 0, io.EOF
	}
	bufLengthInSamples := len(buf) / 4
	writeIndex := 0
	srcChannels := tr.tape.nchannels
	dstChannels := tr.nchannels
	switch srcChannels {
	case 1:
		switch dstChannels {
		case 1:
			framesToWrite := min(bufLengthInSamples, samplesLeft)
			bytesToWrite := framesToWrite * 4
			for writeIndex < bytesToWrite {
				smp := samples[offset]
				offset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
			}
		case 2:
			framesToWrite := min(bufLengthInSamples/2, samplesLeft)
			bytesToWrite := framesToWrite * 8
			for writeIndex < bytesToWrite {
				smp := samples[offset]
				offset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
			}
		}
	case 2:
		switch dstChannels {
		case 1:
			framesToWrite := min(bufLengthInSamples, samplesLeft/2)
			bytesToWrite := framesToWrite * 4
			for writeIndex < bytesToWrite {
				smp := (samples[offset] + samples[offset+1]) / 2.0
				offset += 2
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
			}
		case 2:
			framesToWrite := min(bufLengthInSamples/2, samplesLeft/2)
			bytesToWrite := framesToWrite * 8
			for writeIndex < bytesToWrite {
				smp := samples[offset]
				offset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
				smp = samples[offset]
				offset++
				writeSampleAsFloat32bits(buf, writeIndex, smp)
				writeIndex += 4
			}
		}
	}
	tr.offset = offset
	return writeIndex, nil
}

func MakeTapeReader(tape *Tape, nchannels int) *TapeReader {
	return &TapeReader{
		tape:      tape,
		nchannels: nchannels,
		offset:    0,
	}
}

func init() {
	RegisterMethod[*Tape]("slice", 3, func(vm *VM) error {
		end := int(Pop[Num](vm))
		start := int(Pop[Num](vm))
		t := Top[*Tape](vm)
		nframes := end - start
		slicedTape := &Tape{
			nchannels: t.nchannels,
			nframes:   nframes,
			samples:   t.samples[start*t.nchannels : end*t.nchannels],
		}
		vm.PushVal(slicedTape)
		return nil
	})
}

func makeTape(sr float64, nchannels, nframes int) *Tape {
	samples := make([]Smp, nchannels*(nframes+1))
	return &Tape{
		sr:        sr,
		nchannels: nchannels,
		nframes:   nframes,
		samples:   samples,
	}
}

func pushTape(vm *VM, sr float64, nchannels, nframes int) *Tape {
	tape := makeTape(sr, nchannels, nframes)
	vm.PushVal(tape)
	return tape
}

func init() {
	RegisterWord("tape1", func(vm *VM) error {
		sr := vm.GetFloat(":sr")
		nframes := int(Pop[Num](vm))
		pushTape(vm, sr, 1, nframes)
		return nil
	})

	RegisterWord("tape2", func(vm *VM) error {
		sr := vm.GetFloat(":sr")
		nframes := int(Pop[Num](vm))
		pushTape(vm, sr, 2, nframes)
		return nil
	})
}

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
    void main(void) {
      gl_FragColor = vec4(1.0);
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
	}
	return td, nil
}

func (td *TapeDisplay) Render(tape *Tape, pixelRect Rect, windowSize int, windowOffset int) {
	pixelWidth, pixelHeight := pixelRect.Dx(), pixelRect.Dy()
	if td.tape != tape || td.pixelRect != pixelRect {
		td.tape = tape
		td.pixelRect = pixelRect
		td.vertices = make([][]PointVertex, tape.nchannels)
		for ch := range tape.nchannels {
			td.vertices[ch] = make([]PointVertex, pixelWidth)
			for x := range pixelWidth {
				td.vertices[ch][x].position[0] = float32(x) + 0.5
			}
		}
	}
	channelHeight := float32(pixelHeight) / float32(tape.nchannels)
	channelHeightHalf := channelHeight / 2.0
	incr := float64(windowSize) / float64(pixelWidth)
	readIndex := float64(windowOffset)
	for x := range pixelWidth {
		channelTop := float32(0)
		for ch := range tape.nchannels {
			smp := tape.GetInterpolatedSampleAt(ch, readIndex)
			td.vertices[ch][x].position[1] = channelTop + channelHeightHalf - float32(smp)*channelHeightHalf
			channelTop += channelHeight
		}
		readIndex += incr
	}
	td.program.Use()
	gl.EnableVertexAttribArray(uint32(td.a_position))
	ux := 2.0 / float32(fbSize.X)
	uy := 2.0 / float32(fbSize.Y)
	mScale := mgl.Scale3D(ux, -uy, 1)
	tx := -1.0 + ux*float32(pixelRect.Min.X)
	ty := 1.0 - uy*float32(pixelRect.Min.Y)
	mTranslate := mgl.Translate3D(tx, ty, 0)
	mTransform := mTranslate.Mul4(mScale)
	gl.UniformMatrix4fv(td.u_transform, 1, false, &mTransform[0])
	for ch := range tape.nchannels {
		gl.VertexAttribPointer(
			uint32(td.a_position), 2, gl.FLOAT, false,
			int32(unsafe.Sizeof(PointVertex{})),
			gl.Ptr(&td.vertices[ch][0].position[0]))
		gl.DrawArrays(gl.LINE_STRIP, 0, int32(len(td.vertices[ch])))
	}
	gl.DisableVertexAttribArray(uint32(td.a_position))
}
