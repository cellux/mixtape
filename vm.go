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
	case int:
		return Num(v)
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
	default:
		return x
	}
}

type Equaler interface {
	Equal(other Val) bool
}

func Equal(lhs, rhs Val) bool {
	if lhs == nil && rhs == nil {
		return true
	}
	if l, ok := lhs.(Equaler); ok {
		return l.Equal(rhs)
	}
	return false
}

type Evaler interface {
	Eval(vm *VM) error
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
	valStack    Vec   // values
	envStack    []Map // environments
	markerStack []int // [] markers
	quoteBuffer Vec   // quoted code
	quoteDepth  int   // nesting level {... {.. {..} ..} ...}
}

func CreateVM() (*VM, error) {
	vm := &VM{
		valStack:    make(Vec, 0, 4096),
		envStack:    []Map{rootEnv},
		markerStack: make([]int, 0, 16),
		quoteBuffer: nil,
		quoteDepth:  0,
	}
	return vm, nil
}

func (vm *VM) Reset() {
	vm.valStack = vm.valStack[:0]
	vm.envStack = vm.envStack[:1]
	vm.quoteBuffer = nil
	vm.quoteDepth = 0
}

func (vm *VM) IsQuoting() bool {
	return vm.quoteDepth > 0
}

func (vm *VM) StackSize() int {
	return len(vm.valStack)
}

func (vm *VM) Push(v any) {
	vm.valStack = append(vm.valStack, AsVal(v))
}

func (vm *VM) Top() Val {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		return nil
	}
	return vm.valStack[stacksize-1]
}

func (vm *VM) Pop() Val {
	stacksize := len(vm.valStack)
	if stacksize == 0 {
		return nil
	}
	result := vm.valStack[stacksize-1]
	vm.valStack = vm.valStack[:stacksize-1]
	return result
}

func Pop[T Val](vm *VM) T {
	val := vm.Pop()
	if value, ok := val.(T); ok {
		return value
	} else {
		log.Fatalf("top of value stack has type %T, expected %T", val, *new(T))
		return *new(T)
	}
}

func Top[T Val](vm *VM) T {
	top := vm.Top()
	if value, ok := top.(T); ok {
		return value
	} else {
		log.Fatalf("top of value stack has type %T, expected %T", top, *new(T))
		return *new(T)
	}
}

func (vm *VM) DoDrop() error {
	stackSize := len(vm.valStack)
	if stackSize == 0 {
		return fmt.Errorf("drop: stack underflow")
	}
	vm.Pop()
	return nil
}

func (vm *VM) DoNip() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return fmt.Errorf("nip: stack underflow")
	}
	vm.valStack[stackSize-2] = vm.valStack[stackSize-1]
	vm.valStack = vm.valStack[:stackSize-1]
	return nil
}

func (vm *VM) DoDup() error {
	stackSize := len(vm.valStack)
	if stackSize == 0 {
		return fmt.Errorf("dup: stack underflow")
	}
	vm.Push(vm.Top())
	return nil
}

func (vm *VM) DoSwap() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return fmt.Errorf("swap: stack underflow")
	}
	top := vm.valStack[stackSize-1]
	vm.valStack[stackSize-1] = vm.valStack[stackSize-2]
	vm.valStack[stackSize-2] = top
	return nil
}

func (vm *VM) DoOver() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return fmt.Errorf("over: stack underflow")
	}
	vm.Push(vm.valStack[stackSize-2])
	return nil
}

func (vm *VM) DoMark() error {
	vm.markerStack = append(vm.markerStack, len(vm.valStack))
	return nil
}

func (vm *VM) DoCollect() error {
	if len(vm.markerStack) == 0 {
		return fmt.Errorf("collect: no active marker")
	}
	markerIndex := vm.markerStack[len(vm.markerStack)-1]
	vm.markerStack = vm.markerStack[:len(vm.markerStack)-1]
	stackSize := len(vm.valStack)
	result := make(Vec, stackSize-markerIndex)
	if markerIndex == stackSize {
		vm.Push(result)
	} else {
		copy(result, vm.valStack[markerIndex:])
		vm.valStack[markerIndex] = result
		vm.valStack = vm.valStack[:markerIndex+1]
	}
	return nil
}

func (vm *VM) DoEval() error {
	val := vm.Pop()
	return vm.Eval(val)
}

func (vm *VM) DoIter() error {
	iterable := Pop[Iterable](vm)
	vm.Push(iterable.Iter())
	return nil
}

func (vm *VM) DoNext() error {
	val := vm.Top()
	return vm.Eval(val)
}

func (vm *VM) TopEnv() Map {
	return vm.envStack[len(vm.envStack)-1]
}

func (vm *VM) DoPushEnv() error {
	vm.envStack = append(vm.envStack, make(Map))
	return nil
}

func (vm *VM) DoPopEnv() error {
	stacksize := len(vm.envStack)
	if stacksize == 1 {
		return fmt.Errorf("attempt to pop root env")
	}
	vm.envStack = vm.envStack[:stacksize-1]
	return nil
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

func (vm *VM) GetStream(k any) Stream {
	return Get[Streamable](vm, k).Stream()
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
				case 'p':
					code = append(code, Num(f), Sym("periods"))
				case 's':
					code = append(code, Num(f), Sym("seconds"))
				case 't':
					code = append(code, Num(f), Sym("ticks"))
				default:
					code = append(code, Num(f))
				}
			} else {
				if len(text) > 1 {
					switch text[0] {
					case '@':
						code = append(code, Str(text[1:]), Sym("get"))
					case '>':
						if text == ">=" {
							code = append(code, Sym(text))
						} else {
							code = append(code, Str(text[1:]), Sym("set"))
						}
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

func (vm *VM) Eval(val Val) error {
	if vm.IsQuoting() {
		if val == Sym("{") {
			vm.quoteDepth++
			vm.quoteBuffer = append(vm.quoteBuffer, val)
		} else if val == Sym("}") {
			if vm.quoteDepth == 0 {
				return fmt.Errorf("attempt to unquote when quoteDepth == 0")
			}
			vm.quoteDepth--
			if vm.quoteDepth > 0 {
				vm.quoteBuffer = append(vm.quoteBuffer, val)
			} else {
				vm.Push(vm.quoteBuffer)
				vm.quoteBuffer = nil
			}
		} else {
			vm.quoteBuffer = append(vm.quoteBuffer, val)
		}
		return nil
	}
	if e, ok := val.(Evaler); ok {
		return e.Eval(vm)
	}
	return fmt.Errorf("don't know how to evaluate value of type %T", val)
}

func (vm *VM) ParseAndEval(r io.Reader, filename string) error {
	code, err := vm.Parse(r, filename)
	if err != nil {
		return err
	}
	return vm.Eval(code)
}
