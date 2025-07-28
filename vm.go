package main

import (
	"fmt"
	"io"
	"log"
	"text/scanner"
	"unicode"
)

type Val = any

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
		return Num(v)
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

// VM

type VM struct {
	valStack      Vec   // values
	envStack      []Map // environments
	compileBuffer Vec   // compiled code
}

func CreateVM() (*VM, error) {
	vm := &VM{
		valStack:      make(Vec, 0, 4096),
		envStack:      []Map{rootEnv},
		compileBuffer: nil,
	}
	return vm, nil
}

func (vm *VM) Reset() {
	vm.valStack = vm.valStack[:0]
	vm.envStack = vm.envStack[:1]
	vm.compileBuffer = nil
}

func (vm *VM) IsCompiling() bool {
	return vm.compileBuffer != nil
}

func (vm *VM) StackSize() int {
	return len(vm.valStack)
}

func (vm *VM) PushVal(v any) {
	vm.valStack = append(vm.valStack, AsVal(v))
}

func (vm *VM) TopVal() Val {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		return nil
	}
	return vm.valStack[stacksize-1]
}

func (vm *VM) PopVal() Val {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		return nil
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
	top := vm.TopVal()
	if value, ok := top.(T); ok {
		return value
	} else {
		log.Fatalf("top of value stack has type %T, expected %T", top, *new(T))
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
	if stacksize == 1 {
		log.Fatalf("attempt to pop root env")
	}
	vm.envStack = vm.envStack[:stacksize-1]
}

func (vm *VM) SetVal(k, v any) {
	env := vm.TopEnv()
	env.SetVal(k, v)
}

func (vm *VM) GetVal(k any) Val {
	index := len(vm.envStack) - 1
	for index >= 0 {
		env := vm.envStack[index]
		if val := env.GetVal(k); val != nil {
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
	RegisterWord("stack", func(vm *VM) error {
		vm.PushVal(vm.valStack)
		return nil
	})

	RegisterWord("str", func(vm *VM) error {
		vm.PushVal(fmt.Sprintf("%s", vm.PopVal()))
		return nil
	})

	RegisterWord("dup", func(vm *VM) error {
		vm.PushVal(vm.TopVal())
		return nil
	})

	RegisterWord("swap", func(vm *VM) error {
		stackSize := len(vm.valStack)
		if stackSize < 2 {
			return fmt.Errorf("swap: stack underflow")
		}
		top := vm.valStack[stackSize-1]
		vm.valStack[stackSize-1] = vm.valStack[stackSize-2]
		vm.valStack[stackSize-2] = top
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

	RegisterWord("do", func(vm *VM) error {
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
