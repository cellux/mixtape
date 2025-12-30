package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"strings"
)

type EvalTargetKind int

const (
	evalTargetScript = 's'
	evalTargetFile   = 'f'
)

type EvalTarget struct {
	Kind  EvalTargetKind
	Value string
}

var flags struct {
	LogLevel    string
	SampleRate  int
	BPM         float64 // beats per minute
	TPB         int     // ticks per beat
	EvalTargets []EvalTarget
	Prof        string
}

func SampleRate() int {
	return flags.SampleRate
}

type EvalTargetFlag struct {
	Kind   EvalTargetKind
	Values []string
}

func (f *EvalTargetFlag) String() string {
	if f.Values == nil {
		return ""
	}
	return strings.Join(f.Values, ",")
}

func (f *EvalTargetFlag) Set(val string) error {
	f.Values = append(f.Values, val)
	flags.EvalTargets = append(flags.EvalTargets, EvalTarget{f.Kind, val})
	return nil
}

func runGui(vm *VM, buffers []*Buffer, currentBuffer *Buffer) error {
	app := &App{
		vm:            vm,
		buffers:       buffers,
		currentBuffer: currentBuffer,
	}
	var windowTitle string
	if currentBuffer != nil {
		windowTitle = fmt.Sprintf("mixtape : %s", currentBuffer.Name)
	} else {
		windowTitle = "mixtape"
	}
	return WithGL(windowTitle, app)
}

func withProfileIfNeeded(fn func() error) error {
	if flags.Prof == "" {
		return fn()
	}
	cpuFile, err := os.Create(flags.Prof + ".cpu")
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		cpuFile.Close()
		return err
	}
	defer func() {
		pprof.StopCPUProfile()
		cpuFile.Close()
	}()

	err = fn()

	if memFile, memErr := os.Create(flags.Prof + ".mem"); memErr == nil {
		pprof.WriteHeapProfile(memFile)
		memFile.Close()
	}

	return err
}

func evalAndReport(vm *VM, r io.Reader, name string) error {
	err := vm.ParseAndEval(r, name)
	if vm.evalResult != nil {
		fmt.Println(vm.evalResult)
	}
	return err
}

func runWithArgs(vm *VM, args []string) error {
	if len(flags.EvalTargets) > 0 {
		return withProfileIfNeeded(func() error {
			for _, target := range flags.EvalTargets {
				switch target.Kind {
				case evalTargetScript:
					if err := evalAndReport(vm, strings.NewReader(target.Value), "<script>"); err != nil {
						return err
					}
				case evalTargetFile:
					data, err := os.ReadFile(target.Value)
					if err != nil {
						return err
					}
					if err := evalAndReport(vm, bytes.NewReader(data), target.Value); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}

	buffers := []*Buffer{}
	var currentBuffer *Buffer

	for _, arg := range args {
		data, err := os.ReadFile(arg)
		if err != nil {
			return err
		}
		path := arg
		buf := CreateBuffer(buffers, path, data)
		buffers = append(buffers, buf)
		currentBuffer = buf
	}

	if len(buffers) == 0 {
		currentBuffer = NewScratchBuffer()
		buffers = append(buffers, currentBuffer)
	}

	return runGui(vm, buffers, currentBuffer)
}

func setDefaults(vm *VM) {
	vm.SetVal(":bpm", flags.BPM)
	vm.SetVal(":tpb", flags.TPB)

	beatsPerSecond := flags.BPM / 60.0
	framesPerBeat := float64(SampleRate()) / beatsPerSecond
	vm.SetVal(":nf", int(framesPerBeat))
}

func main() {
	var vm *VM
	var err error
	flag.StringVar(&flags.LogLevel, "loglevel", "info", "Log level")
	flag.IntVar(&flags.SampleRate, "sr", 48000, "Sample rate")
	flag.Float64Var(&flags.BPM, "bpm", 120, "Beats per minute")
	flag.IntVar(&flags.TPB, "tpb", 96, "Ticks per beat")
	flag.Var(&EvalTargetFlag{Kind: evalTargetFile}, "f", "File to evaluate")
	flag.Var(&EvalTargetFlag{Kind: evalTargetScript}, "e", "Script to evaluate")
	flag.StringVar(&flags.Prof, "prof", "", "Profile output file prefix (writes <prefix>.cpu and <prefix>.mem)")
	flag.Parse()
	if err := InitLogger(flags.LogLevel); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	vm, err = CreateVM()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vm initialization error: %s", err)
		os.Exit(1)
	}
	setDefaults(vm)
	prelude, err := assets.ReadFile("assets/prelude.tape")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load prelude from embed.FS: %s", err)
		os.Exit(1)
	}
	err = vm.ParseAndEval(bytes.NewReader(prelude), "<prelude>")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while parsing the prelude: %s", err)
		os.Exit(1)
	}
	err = runWithArgs(vm, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
