package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"text/scanner"
	"unicode"
)

type Smp = float64

type Val interface{}

type Num float64
type Str string
type Sym string
type Fun func(vm *VM) error
type Vec []Val
type Map map[Val]Val

type SampleIterator func() Smp

type SampleIteratorProvider interface {
	GetSampleIterator() SampleIterator
}

func GetSampleIterator(val Val) SampleIterator {
	if provider, ok := val.(SampleIteratorProvider); ok {
		return provider.GetSampleIterator()
	}
	log.Fatalf("value of type %T does not implement SampleIteratorProvider", val)
	return nil
}

type MessageHandler interface {
	GetMessageHandler(msg string, nargs int) Fun
}

func (n Num) String() string {
	return fmt.Sprintf("%g", float64(n))
}

func (n Num) GetSampleIterator() SampleIterator {
	return func() Smp {
		return Smp(n)
	}
}

func (n Num) GetMessageHandler(msg string, nargs int) Fun {
	if nargs == 1 {
		switch msg {
		case "+":
			return func(vm *VM) error {
				rhs := vm.PopNum()
				lhs := vm.PopNum()
				vm.PushVal(lhs + rhs)
				return nil
			}
		case "-":
			return func(vm *VM) error {
				rhs := vm.PopNum()
				lhs := vm.PopNum()
				vm.PushVal(lhs - rhs)
				return nil
			}
		case "*":
			return func(vm *VM) error {
				rhs := vm.PopNum()
				lhs := vm.PopNum()
				vm.PushVal(lhs * rhs)
				return nil
			}
		case "/":
			return func(vm *VM) error {
				rhs := vm.PopNum()
				lhs := vm.PopNum()
				vm.PushVal(lhs / rhs)
				return nil
			}
		}
	}
	return nil
}

func (s Str) String() string {
	return string(s)
}

func (s Sym) String() string {
	return string(s)
}

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func (m Map) String() string {
	return fmt.Sprintf("%v", map[Val]Val(m))
}

type VM struct {
	valStack      Vec   // values
	envStack      []Map // environments
	compileBuffer Vec   // compiled code
}

func (vm *VM) IsCompiling() bool {
	return vm.compileBuffer != nil
}

func (vm *VM) PushVal(v Val) {
	vm.valStack = append(vm.valStack, v)
}

func (vm *VM) PopVal() Val {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		log.Fatalf("value stack underflow")
	}
	result := vm.valStack[stacksize-1]
	vm.valStack = vm.valStack[:stacksize-1]
	return result
}

func Pop[T Val](vm *VM) T {
	val := vm.PopVal()
	if value, ok := val.(T); ok {
		return value
	} else {
		log.Fatalf("top of value stack has type %T, expected %T", val, *new(T))
		return *new(T)
	}
}

func (vm *VM) PopNum() Num {
	return Pop[Num](vm)
}

func (vm *VM) ValAt(index int) Val {
	actualIndex := index
	if index < 0 {
		actualIndex = len(vm.valStack) + index
	}
	if actualIndex >= len(vm.valStack) {
		log.Fatalf("value stack index out of bounds: %d", index)
	}
	return vm.valStack[actualIndex]
}

func At[T Val](vm *VM, index int) T {
	val := vm.ValAt(index)
	if value, ok := val.(T); ok {
		return value
	} else {
		log.Fatalf("value at stack index %d has type %T, expected %T", index, val, *new(T))
		return *new(T)
	}
}

func (vm *VM) NumAt(index int) Num {
	return At[Num](vm, index)
}

func (vm *VM) TopEnv() Map {
	return vm.envStack[len(vm.envStack)-1]
}

func (vm *VM) PushEnv() {
	vm.envStack = append(vm.envStack, make(Map))
}

func (vm *VM) PopEnv() {
	stacksize := len(vm.envStack)
	if stacksize == 0 {
		panic("env stack underflow")
	}
	vm.envStack = vm.envStack[:stacksize-1]
}

func AsVal(x any) Val {
	switch v := x.(type) {
	case Num, Str, Sym, Fun, Vec, Map:
		return v
	case int:
		return Num(float64(v))
	case float64:
		return Num(float64(v))
	case string:
		return Str(v)
	case func(vm *VM) error:
		return Fun(v)
	case []Val:
		return Vec(v)
	case map[Val]Val:
		return Map(v)
	default:
		log.Fatalf("AsVal: don't know how to convert value of type %T into Val", x)
		return nil
	}
}

func (vm *VM) SetVal(k, v any) {
	env := vm.TopEnv()
	key := AsVal(k)
	val := AsVal(v)
	env[key] = val
}

func (vm *VM) GetVal(k any) Val {
	key := AsVal(k)
	index := len(vm.envStack) - 1
	for index >= 0 {
		env := vm.envStack[index]
		if val, ok := env[key]; ok {
			return val
		}
		index--
	}
	return nil
}

func Get[T Val](vm *VM, k any) T {
	val := vm.GetVal(k)
	if value, ok := val.(T); ok {
		return value
	} else {
		log.Fatalf("value at key %s is of type %T, expected %T", k, val, *new(T))
		return *new(T)
	}
}

func (vm *VM) GetNum(k any) Num {
	return Get[Num](vm, k)
}

func (vm *VM) GetFloat(k any) float64 {
	return float64(Get[Num](vm, k))
}

func (vm *VM) GetInt(k any) int {
	return int(Get[Num](vm, k))
}

func wordStackPrint(vm *VM) error {
	fmt.Printf("%s\n", vm.valStack)
	return nil
}

func wordValuePopAndPrint(vm *VM) error {
	fmt.Printf("%s\n", vm.PopVal())
	return nil
}

func wordPushEnv(vm *VM) error {
	vm.PushEnv()
	return nil
}

func wordPopEnv(vm *VM) error {
	vm.PopEnv()
	return nil
}

func wordSet(vm *VM) error {
	k := vm.PopVal()
	v := vm.PopVal()
	vm.SetVal(k, v)
	return nil
}

func wordGet(vm *VM) error {
	k := vm.PopVal()
	v := vm.GetVal(k)
	vm.PushVal(v)
	return nil
}

func wordExecute(vm *VM) error {
	word := vm.PopVal()
	return vm.Execute(word)
}

func wordDispatch(vm *VM) error {
	msg := string(Pop[Str](vm))
	handler := vm.FindMessageHandler(msg)
	if handler != nil {
		return handler(vm)
	}
	return fmt.Errorf("unhandled message: %s", msg)
}

func wordCompile(vm *VM) error {
	vm.compileBuffer = make(Vec, 0, 256)
	return nil
}

func wordSeconds(vm *VM) error {
	n := vm.PopNum()
	sr := vm.GetNum(":sr")
	vm.PushVal(n * sr)
	return nil
}

func wordBeats(vm *VM) error {
	n := vm.PopNum()
	sr := vm.GetNum(":sr")
	bpm := vm.GetNum(":bpm")
	beatsPerSecond := bpm / 60.0
	framesPerBeat := sr / beatsPerSecond
	vm.PushVal(n * framesPerBeat)
	return nil
}

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

func NewVM() *VM {
	rootEnv := make(Map)
	vm := &VM{
		valStack: make(Vec, 0, 4096),
		envStack: []Map{rootEnv},
	}
	vm.SetVal(":bpm", 120)
	vm.SetVal(":sr", 48000)
	vm.SetVal(":length", 48000)
	vm.SetVal(":freq", 440)
	vm.SetVal(":phase", 0)
	vm.SetVal("[", wordCompile)
	vm.SetVal("{", wordPushEnv)
	vm.SetVal("}", wordPopEnv)
	vm.SetVal(".", wordValuePopAndPrint)
	vm.SetVal("ps", wordStackPrint)
	vm.SetVal("get", wordGet)
	vm.SetVal("set", wordSet)
	vm.SetVal("execute", wordExecute)
	vm.SetVal("dispatch", wordDispatch)
	vm.SetVal("seconds", wordSeconds)
	vm.SetVal("beats", wordBeats)
	vm.SetVal("tape1", wordTape1)
	vm.SetVal("tape2", wordTape2)
	return vm
}

func (vm *VM) Parse(r io.Reader, filename string) (Vec, error) {
	var s scanner.Scanner
	s.Init(r)
	s.IsIdentRune = func(ch rune, i int) bool {
		if unicode.IsSpace(ch) || unicode.IsControl(ch) {
			return false
		}
		if ch == '{' || ch == '}' {
			return false
		}
		if ch == '[' || ch == ']' {
			return false
		}
		if i == 0 {
			if ch == '"' {
				return false
			}
		}
		return true
	}
	s.Filename = filename
	var code = make(Vec, 0, 16384)
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.Char, scanner.String, scanner.RawString:
			code = append(code, Str(s.TokenText()))
		case '{', '}', '[', ']':
			code = append(code, Sym(string(tok)))
		case scanner.Ident:
			text := s.TokenText()
			var f float64
			_, err := fmt.Sscanf(text, "%g", &f)
			if err == nil {
				var nominator, denominator int
				_, err := fmt.Sscanf(text, "%d/%d", &nominator, &denominator)
				if err == nil {
					f = float64(nominator) / float64(denominator)
				}
				switch text[len(text)-1] {
				case 'b':
					code = append(code, Num(f), Sym("beats"))
				case 's':
					code = append(code, Num(f), Sym("seconds"))
				default:
					code = append(code, Num(f))
				}
			} else {
				if len(text) > 1 {
					switch text[0] {
					case '>':
						code = append(code, Str(text[1:]), Sym("set"))
					case '.':
						code = append(code, Str(text[1:]), Sym("dispatch"))
					default:
						code = append(code, Sym(text))
					}
				} else {
					code = append(code, Sym(text))
				}
			}
		default:
			return nil, fmt.Errorf("parse error at %s: %s", s.Position, s.TokenText())
		}
	}
	return code, nil
}

func (vm *VM) FindMessageHandler(msg string) Fun {
	nargs := 0
	index := len(vm.valStack) - 1
	for index >= 0 {
		val := vm.valStack[index]
		if mh, ok := val.(MessageHandler); ok {
			handler := mh.GetMessageHandler(msg, nargs)
			if handler != nil {
				return handler
			}
		}
		index--
		nargs++
	}
	return nil
}

func (vm *VM) Execute(val Val) error {
	if vm.IsCompiling() {
		if val == Sym("]") {
			vm.PushVal(vm.compileBuffer)
			vm.compileBuffer = nil
			return nil
		} else {
			vm.compileBuffer = append(vm.compileBuffer, val)
			return nil
		}
	}
	switch value := val.(type) {
	case Fun:
		return value(vm)
	case Num, Str:
		vm.PushVal(value)
	case Sym:
		name := string(value)
		word := vm.GetVal(name)
		if word != nil {
			return vm.Execute(word)
		}
		handler := vm.FindMessageHandler(name)
		if handler != nil {
			return handler(vm)
		}
		return fmt.Errorf("undefined word: %s", name)
	case Vec:
		for _, val := range value {
			err := vm.Execute(val)
			if err != nil {
				return err
			}
		}
	default:
		log.Fatalf("don't know how to execute value of type %T", value)
	}
	return nil
}

func (vm *VM) ParseAndExecute(r io.Reader, filename string) error {
	code, err := vm.Parse(r, filename)
	if err != nil {
		return err
	}
	return vm.Execute(code)
}
