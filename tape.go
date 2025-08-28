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
	"math"
	"os"
	"path/filepath"
	"unsafe"
)

type Tape struct {
	nchannels int
	nframes   int
	samples   []Smp
}

func (t *Tape) String() string {
	return fmt.Sprintf("Tape(nchannels=%d nframes=%d)", t.nchannels, t.nframes)
}

func (t *Tape) Stream() Stream {
	return makeStream(t.nchannels, t.nframes,
		func(yield func(Frame) bool) {
			index := 0
			for range t.nframes {
				if !yield(t.samples[index : index+t.nchannels]) {
					return
				}
				index += t.nchannels
			}
		})
}

func (t *Tape) GetInterpolatedSampleAt(channel int, frame float64) Smp {
	frameIndexLo := math.Floor(frame)
	sampleIndexLo := channel + int(frameIndexLo)*t.nchannels
	smpLo := t.samples[sampleIndexLo]
	if int(frameIndexLo) == t.nframes-1 {
		return smpLo
	}
	smpHi := t.samples[sampleIndexLo+1]
	frameIndexDelta := frame - frameIndexLo
	return smpLo + (smpHi-smpLo)*frameIndexDelta
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
	RegisterMethod[*Tape]("nf", 1, func(vm *VM) error {
		t := Pop[*Tape](vm)
		vm.Push(t.nframes)
		return nil
	})

	RegisterMethod[*Tape]("join", 2, func(vm *VM) error {
		rhs := Pop[*Tape](vm)
		lhs := Pop[*Tape](vm)
		if lhs.nchannels != rhs.nchannels {
			return fmt.Errorf("join: lhs and rhs must have the same number of channels, got lhs=%d, rhs=%d", lhs.nchannels, rhs.nchannels)
		}
		nf := lhs.nframes + rhs.nframes
		t := pushTape(vm, lhs.nchannels, nf)
		leftEnd := lhs.nframes * lhs.nchannels
		copy(t.samples[:leftEnd], lhs.samples)
		copy(t.samples[leftEnd:], rhs.samples)
		return nil
	})

	RegisterMethod[*Tape]("shift", 2, func(vm *VM) error {
		amount := Pop[Num](vm)
		t := Top[*Tape](vm)
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
		ratio := float64(Pop[Num](vm))
		if ratio <= 0 {
			return fmt.Errorf("resample: invalid ratio: %f", ratio)
		}
		converterType := int(Pop[Num](vm))
		if converterType < 0 || converterType > 4 {
			return fmt.Errorf("resample: invalid converterType: %d - must be between 0..4", converterType)
		}
		t := Pop[*Tape](vm)
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

func loadAndPushTape(vm *VM, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sr := SampleRate()
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
		err = vm.ParseAndEval(f, path)
		if err != nil {
			return err
		}
		val := vm.Top()
		if tape, ok := val.(*Tape); ok {
			err := tape.WriteToWav(wavPath)
			if err != nil {
				return err
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
				tape.samples[i] = float64(resampledBuf[i])
			}
		} else {
			tape := pushTape(vm, nchannels, nframes)
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
		mp3SR := decoder.SampleRate()
		if mp3SR != sr {
			var startTime float64
			logger.Debug("decoding mp3 file", "path", path)
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
				tape.samples[i] = float64(resampledBuf[i])
			}
		} else {
			var startTime float64
			logger.Debug("decoding mp3 file", "path", path)
			startTime = GetTime()
			var sample int16
			tape := pushTape(vm, nchannels, nframes)
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
			logger.Debug("decoded mp3 file", "path", path, "seconds", GetTime()-startTime)
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
		logger.Debug("playing finished")
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
	RegisterMethod[*Tape]("slice", 2, func(vm *VM) error {
		end := int(Pop[Num](vm))
		start := int(Pop[Num](vm))
		t := Top[*Tape](vm)
		vm.Push(t.Slice(start, end))
		return nil
	})

	RegisterMethod[*Tape]("+@", 3, func(vm *VM) error {
		offset := int(Pop[Num](vm))
		rhs := Pop[*Tape](vm)
		lhs := Top[*Tape](vm)
		nchannels := lhs.nchannels
		end := offset + rhs.nframes
		if lhs.nframes < end {
			extraFramesNeeded := end - lhs.nframes
			lhs.samples = append(lhs.samples, make([]Smp, extraFramesNeeded*nchannels)...)
			lhs.nframes += extraFramesNeeded
		}
		s := rhs.Stream().AdaptChannels(nchannels)
		writeIndex := offset * nchannels
		for frame := range s.seq {
			for i := range nchannels {
				lhs.samples[writeIndex] += frame[i]
				writeIndex++
			}
		}
		return nil
	})
}

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

func init() {
	RegisterWord("tape1", func(vm *VM) error {
		nframes := int(Pop[Num](vm))
		pushTape(vm, 1, nframes)
		return nil
	})

	RegisterWord("tape2", func(vm *VM) error {
		nframes := int(Pop[Num](vm))
		pushTape(vm, 2, nframes)
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
