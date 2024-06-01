package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"text/scanner"
	"unicode"
)

type Val interface{}

type Num float64
type Str string
type Sym string
type Fun func(vm *VM) error
type Vec []Val
type Map map[Val]Val

const (
	True  = Num(-1)
	False = Num(0)
)

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
	case bool:
		if v {
			return True
		} else {
			return False
		}
	case func(vm *VM) error:
		return Fun(v)
	case []Val:
		return Vec(v)
	case map[Val]Val:
		return Map(v)
	default:
		return x
	}
}

var rootEnv = make(Map)

func RegisterNum(name string, num Num) {
	rootEnv.SetVal(name, num)
}

func RegisterWord(name string, fun Fun) {
	rootEnv.SetVal(name, fun)
}

type Method struct {
	nargs int
	fun   Fun
}

type MethodMap map[string][]Method

func (mm MethodMap) RegisterMethod(name string, nargs int, fun Fun) {
	if _, ok := mm[name]; !ok {
		mm[name] = make([]Method, 0, 8)
	}
	mm[name] = append(mm[name], Method{nargs, fun})
}

func (mm MethodMap) FindMethod(name string, nargs int) Fun {
	if _, ok := mm[name]; !ok {
		return nil
	}
	for _, method := range mm[name] {
		if method.nargs == nargs {
			return method.fun
		}
	}
	return nil
}

type TypeMethodMap map[reflect.Type]MethodMap

var typeMethods = make(TypeMethodMap)

func RegisterMethod[T any](name string, nargs int, fun Fun) {
	t := reflect.TypeFor[T]()
	if _, ok := typeMethods[t]; !ok {
		typeMethods[t] = make(MethodMap)
	}
	typeMethods[t].RegisterMethod(name, nargs, fun)
}

func FindMethod(val Val, name string, nargs int) Fun {
	t := reflect.TypeOf(val)
	if _, ok := typeMethods[t]; !ok {
		return nil
	}
	return typeMethods[t].FindMethod(name, nargs)
}

// Num

func init() {
	RegisterMethod[Num]("=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs == rhs)
		return nil
	})

	RegisterMethod[Num]("!=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs != rhs)
		return nil
	})

	RegisterMethod[Num]("not", 1, func(vm *VM) error {
		arg := Pop[Num](vm)
		vm.PushVal(arg == 0)
		return nil
	})

	RegisterMethod[Num]("assert", 1, func(vm *VM) error {
		n := Pop[Num](vm)
		if n == False {
			return fmt.Errorf("assertion failed")
		}
		return nil
	})

	RegisterMethod[Num]("+", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs + rhs)
		return nil
	})

	RegisterMethod[Num]("-", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs - rhs)
		return nil
	})

	RegisterMethod[Num]("*", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs * rhs)
		return nil
	})

	RegisterMethod[Num]("/", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs / rhs)
		return nil
	})

	RegisterMethod[Num]("%", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(math.Mod(float64(lhs), float64(rhs)))
		return nil
	})

	RegisterMethod[Num]("<", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs < rhs)
		return nil
	})

	RegisterMethod[Num]("<=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs <= rhs)
		return nil
	})

	RegisterMethod[Num](">=", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs >= rhs)
		return nil
	})

	RegisterMethod[Num](">", 2, func(vm *VM) error {
		rhs := Pop[Num](vm)
		lhs := Pop[Num](vm)
		vm.PushVal(lhs > rhs)
		return nil
	})
}

func (n Num) String() string {
	return fmt.Sprintf("%g", n)
}

func (n Num) GetSampleIterator() SampleIterator {
	return func() Smp {
		return Smp(n)
	}
}

// Str

func scanFloat(text string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(text, "%g", &f)
	if err == nil {
		var nominator, denominator int
		_, err = fmt.Sscanf(text, "%d/%d", &nominator, &denominator)
		if err == nil {
			return float64(nominator) / float64(denominator), nil
		} else {
			return f, nil
		}
	}
	return 0, fmt.Errorf("cannot parse float: %s", text)
}

func init() {
	RegisterMethod[Str]("num", 1, func(vm *VM) error {
		arg := Pop[Str](vm)
		f, err := scanFloat(string(arg))
		if err != nil {
			return err
		}
		vm.PushVal(f)
		return nil
	})

	RegisterMethod[Str]("=", 2, func(vm *VM) error {
		rhs := Pop[Str](vm)
		lhs := Pop[Str](vm)
		vm.PushVal(lhs == rhs)
		return nil
	})
}

func (s Str) String() string {
	return string(s)
}

// Sym

func (s Sym) String() string {
	return string(s)
}

// Vec

func (v Vec) String() string {
	return fmt.Sprintf("%v", []Val(v))
}

func init() {
	RegisterMethod[Vec]("len", 1, func(vm *VM) error {
		arg := Pop[Vec](vm)
		vm.PushVal(len(arg))
		return nil
	})
}

// Map

func (m Map) String() string {
	return fmt.Sprintf("%v", map[Val]Val(m))
}

func (m Map) SetVal(k, v any) {
	key := AsVal(k)
	val := AsVal(v)
	m[key] = val
}

// VM

type VM struct {
	valStack      Vec   // values
	envStack      []Map // environments
	compileBuffer Vec   // compiled code
}

func NewVM() *VM {
	return &VM{
		valStack:      make(Vec, 0, 4096),
		envStack:      []Map{rootEnv},
		compileBuffer: nil,
	}
}

func (vm *VM) Reset() {
	vm.valStack = vm.valStack[:0]
	vm.envStack = vm.envStack[:1]
	vm.compileBuffer = nil
}

func (vm *VM) IsCompiling() bool {
	return vm.compileBuffer != nil
}

func (vm *VM) PushVal(v any) {
	vm.valStack = append(vm.valStack, AsVal(v))
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

func Top[T Val](vm *VM) T {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		log.Fatalf("value stack underflow")
	}
	val := vm.valStack[stacksize-1]
	if value, ok := val.(T); ok {
		return value
	} else {
		log.Fatalf("top of value stack has type %T, expected %T", val, *new(T))
		return *new(T)
	}
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

func (vm *VM) SetVal(k, v any) {
	env := vm.TopEnv()
	env.SetVal(k, v)
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

func (vm *VM) Parse(r io.Reader, filename string) (Vec, error) {
	var s scanner.Scanner
	s.Init(r)
	s.IsIdentRune = func(ch rune, i int) bool {
		if unicode.IsSpace(ch) || unicode.IsControl(ch) {
			return false
		}
		if ch == '(' || ch == ')' {
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
			if ch == '#' {
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
			text := s.TokenText()
			code = append(code, Str(text[1:len(text)-1]))
		case '#':
			for {
				ch := s.Next()
				if ch == '\n' || ch == scanner.EOF {
					break
				}
			}
		case '(', ')', '{', '}', '[', ']':
			code = append(code, Sym(string(tok)))
		case scanner.Ident:
			text := s.TokenText()
			f, err := scanFloat(text)
			if err == nil {
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
						if text == ">=" {
							code = append(code, Sym(text))
						} else {
							code = append(code, Str(text[1:]), Sym("set"))
						}
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

func (vm *VM) FindMethod(name string) Fun {
	nargs := 1
	index := len(vm.valStack) - 1
	for index >= 0 {
		val := vm.valStack[index]
		method := FindMethod(val, name, nargs)
		if method != nil {
			return method
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
		} else {
			vm.compileBuffer = append(vm.compileBuffer, val)
		}
		return nil
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
		method := vm.FindMethod(name)
		if method != nil {
			return method(vm)
		}
		return fmt.Errorf("word or method not found: %s", name)
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

func init() {
	RegisterNum(":bpm", 120)
	RegisterNum(":sr", 48000)
	RegisterNum(":freq", 440)
	RegisterNum(":phase", 0)
	RegisterNum(":width", 0.5)

	RegisterWord("stack", func(vm *VM) error {
		vm.PushVal(vm.valStack)
		return nil
	})

	RegisterWord("str", func(vm *VM) error {
		val := vm.PopVal()
		vm.PushVal(fmt.Sprintf("%s", val))
		return nil
	})

	RegisterWord("dup", func(vm *VM) error {
		stacksize := len(vm.valStack)
		if stacksize == 0 {
			log.Fatalf("value stack underflow")
		}
		topVal := vm.valStack[stacksize-1]
		vm.PushVal(topVal)
		return nil
	})

	RegisterWord("ps", func(vm *VM) error {
		fmt.Printf("%s\n", vm.valStack)
		return nil
	})

	RegisterWord(".", func(vm *VM) error {
		fmt.Printf("%s\n", vm.PopVal())
		return nil
	})

	RegisterWord("(", func(vm *VM) error {
		vm.PushEnv()
		return nil
	})

	RegisterWord(")", func(vm *VM) error {
		vm.PopEnv()
		return nil
	})

	RegisterWord("set", func(vm *VM) error {
		k := vm.PopVal()
		v := vm.PopVal()
		vm.SetVal(k, v)
		return nil
	})

	RegisterWord("get", func(vm *VM) error {
		k := vm.PopVal()
		v := vm.GetVal(k)
		vm.PushVal(v)
		return nil
	})

	RegisterWord("execute", func(vm *VM) error {
		word := vm.PopVal()
		return vm.Execute(word)
	})

	RegisterWord("dispatch", func(vm *VM) error {
		name := string(Pop[Str](vm))
		method := vm.FindMethod(name)
		if method != nil {
			return method(vm)
		}
		return fmt.Errorf("method not found: %s", name)
	})

	RegisterWord("[", func(vm *VM) error {
		vm.compileBuffer = make(Vec, 0, 64)
		return nil
	})

	RegisterWord("seconds", func(vm *VM) error {
		n := Pop[Num](vm)
		sr := vm.GetNum(":sr")
		vm.PushVal(n * sr)
		return nil
	})

	RegisterWord("beats", func(vm *VM) error {
		n := Pop[Num](vm)
		sr := vm.GetNum(":sr")
		bpm := vm.GetNum(":bpm")
		beatsPerSecond := bpm / 60.0
		framesPerBeat := sr / beatsPerSecond
		vm.PushVal(n * framesPerBeat)
		return nil
	})
}
