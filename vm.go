package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"text/scanner"
	"unicode"
)

type Val interface {
	fmt.Stringer
	getVal() Val
}

const (
	True  = Num(-1)
	False = Num(0)
)

func AsVal(x any) Val {
	if x == nil {
		return Nil
	}
	if v, ok := x.(Val); ok {
		return v.getVal()
	}
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
		return makeErr(fmt.Errorf("cannot convert value of type %T to Val", x))
	}
}

type Equaler interface {
	Equal(other Val) bool
}

func Equal(lhs, rhs Val) bool {
	if lhs == Nil {
		return rhs == Nil
	}
	if l, ok := lhs.(Equaler); ok {
		return l.Equal(rhs)
	}
	return false
}

type Evaler interface {
	Val
	Eval(vm *VM) error
}

type Iterable interface {
	Val
	Iter() Fun
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
	valStack    Vec           // values
	envStack    []Map         // environments
	markerStack []int         // [ indices in valStack
	quoteBuffer Vec           // quoted code
	quoteDepth  int           // nesting level {... {.. {..} ..} ...}
	tokenStack  Box[[]*Token] // call stack of currently executing tokens

	// Evaluation lifecycle
	//
	// isEvaluating is used for quick checks inside tight loops.
	// cancelCh is closed to request cancellation of the current evaluation.
	// doneCh is closed when the current evaluation finishes (success, error, or cancellation).
	//
	// These channels are per-evaluation run, guarded by evalMu.
	evalMu       sync.Mutex
	isEvaluating Box[bool]
	cancelCh     chan struct{}
	doneCh       chan struct{}

	evalResult           Val   // top of stack after a successful evaluation
	errResult            error // last evaluation error
	tapeProgressCallback func(t *Tape, nftotal, nfdone int)
}

func CreateVM() (*VM, error) {
	vm := &VM{
		valStack:    make(Vec, 0, 4096),
		envStack:    []Map{rootEnv},
		markerStack: make([]int, 0, 16),
	}
	return vm, nil
}

type VMStackState struct {
	valStackSize    int
	envStackSize    int
	markerStackSize int
}

func (vm *VM) SaveStackState() *VMStackState {
	return &VMStackState{
		valStackSize:    len(vm.valStack),
		envStackSize:    len(vm.envStack),
		markerStackSize: len(vm.markerStack),
	}
}

func (vm *VM) RestoreStackState(state *VMStackState) {
	if state.valStackSize < len(vm.valStack) {
		vm.valStack = vm.valStack[:state.valStackSize]
	}
	if state.envStackSize < len(vm.envStack) {
		vm.envStack = vm.envStack[:state.envStackSize]
	}
	if state.markerStackSize < len(vm.markerStack) {
		vm.markerStack = vm.markerStack[:state.markerStackSize]
	}
}

func (vm *VM) Reset() {
	vm.valStack = vm.valStack[:0]
	vm.envStack = vm.envStack[:1]
	vm.markerStack = vm.markerStack[:0]
	vm.quoteBuffer = nil
	vm.quoteDepth = 0
	vm.tokenStack.Set(nil)
	vm.isEvaluating.Set(false)

	// Do not close doneCh/cancelCh here: those are lifecycle signals managed by ParseAndEval.
}

func (vm *VM) IsEvaluating() bool {
	return vm.isEvaluating.Get()
}

func (vm *VM) CancelEvaluation() {
	// Fast path: flip the flag so cooperative checks can bail out quickly.
	vm.isEvaluating.Set(false)

	// Signal cancellation (idempotent) and then wait for the evaluation to finish.
	vm.evalMu.Lock()
	cancelCh := vm.cancelCh
	doneCh := vm.doneCh
	vm.evalMu.Unlock()

	if cancelCh != nil {
		close(cancelCh)
	}
	if doneCh != nil {
		<-doneCh
	}
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

func Pop[T Val](vm *VM) (T, error) {
	val := vm.Pop()
	if value, ok := val.(T); ok {
		return value, nil
	} else {
		var zeroT T
		return zeroT, vm.Errorf("value at top of stack has type %T, expected %T", val, zeroT)
	}
}

func Top[T Val](vm *VM) (T, error) {
	top := vm.Top()
	if value, ok := top.(T); ok {
		return value, nil
	} else {
		var zeroT T
		return zeroT, vm.Errorf("value at top of stack has type %T, expected %T", top, zeroT)
	}
}

func (vm *VM) DoDrop() error {
	stackSize := len(vm.valStack)
	if stackSize == 0 {
		return vm.Errorf("drop: stack underflow")
	}
	vm.Pop()
	return nil
}

func (vm *VM) DoNip() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return vm.Errorf("nip: stack underflow")
	}
	vm.valStack[stackSize-2] = vm.valStack[stackSize-1]
	vm.valStack = vm.valStack[:stackSize-1]
	return nil
}

func (vm *VM) DoDup() error {
	stackSize := len(vm.valStack)
	if stackSize == 0 {
		return vm.Errorf("dup: stack underflow")
	}
	vm.Push(vm.Top())
	return nil
}

func (vm *VM) DoSwap() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return vm.Errorf("swap: stack underflow")
	}
	top := vm.valStack[stackSize-1]
	vm.valStack[stackSize-1] = vm.valStack[stackSize-2]
	vm.valStack[stackSize-2] = top
	return nil
}

func (vm *VM) DoOver() error {
	stackSize := len(vm.valStack)
	if stackSize < 2 {
		return vm.Errorf("over: stack underflow")
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
		return vm.Errorf("collect: no active marker")
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

func (vm *VM) DoQuote() error {
	if vm.quoteDepth != 0 {
		return vm.Errorf("attempt to evaluate { when quoteDepth=%d", vm.quoteDepth)
	}
	vm.quoteBuffer = make(Vec, 0, 64)
	vm.quoteDepth++
	return nil
}

func (vm *VM) DoEval() error {
	val := vm.Pop()
	return vm.Eval(val)
}

func (vm *VM) DoIter() error {
	iterable, err := Pop[Iterable](vm)
	if err != nil {
		return err
	}
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
		return vm.Errorf("attempt to pop root env")
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

func Get[T Val](vm *VM, k any) (T, error) {
	val := vm.GetVal(k)
	if val == nil {
		var zeroT T
		return zeroT, vm.Errorf("key not found: %v", k)
	}
	if value, ok := val.(T); ok {
		return value, nil
	}
	var zeroT T
	return zeroT, vm.Errorf("value at key %v is of type %T, expected %T", k, val, zeroT)
}

func (vm *VM) GetNum(k any) (Num, error) {
	return Get[Num](vm, k)
}

func (vm *VM) GetFloat(k any) (float64, error) {
	n, err := vm.GetNum(k)
	return float64(n), err
}

func (vm *VM) GetInt(k any) (int, error) {
	n, err := vm.GetNum(k)
	return int(n), err
}

func (vm *VM) GetStream(k any) (Stream, error) {
	return streamFromVal(vm.GetVal(k))
}

type Token struct {
	v      Val
	pos    scanner.Position
	length int
}

func (t *Token) getVal() Val {
	return t.v
}

func (vm *VM) pushToken(t *Token) {
	vm.tokenStack.Update(func(stack []*Token) []*Token {
		return append(stack, t)
	})
}

func (vm *VM) popToken() {
	vm.tokenStack.Update(func(stack []*Token) []*Token {
		if len(stack) == 0 {
			return stack
		}
		return stack[:len(stack)-1]
	})
}

func (vm *VM) CurrentToken() *Token {
	stack := vm.tokenStack.Get()
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

func (t *Token) Eval(vm *VM) error {
	vm.pushToken(t)
	err := vm.Eval(t.getVal())
	vm.popToken()
	return err
}

func (t *Token) String() string {
	return t.getVal().String()
}

func (t *Token) Equal(other Val) bool {
	return Equal(t.getVal(), other.getVal())
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
			if ch == ';' {
				return false
			}
		}
		return true
	}
	s.Filename = filename
	var code = make(Vec, 0, 16384)
	noteRegex := regexp.MustCompile(`(?i)^[cdefgab][#-][0-9]$`)
	appendTokens := func(text string, vs ...Val) {
		pos := s.Position
		length := len(text)
		for _, v := range vs {
			code = append(code, &Token{v: v, pos: pos, length: length})
		}
	}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.Char, scanner.String, scanner.RawString:
			text := s.TokenText()
			appendTokens(text, Str(text[1:len(text)-1]))
		case ';':
			for {
				ch := s.Next()
				if ch == '\n' || ch == scanner.EOF {
					break
				}
			}
		case '(', ')', '{', '}', '[', ']':
			appendTokens(string(tok), Sym(string(tok)))
		case scanner.Ident:
			text := s.TokenText()
			f, err := scanFloat(text)
			if err == nil {
				switch text[len(text)-1] {
				case 'b':
					appendTokens(text, Num(f), Sym("beats"))
				case 'p':
					appendTokens(text, Num(f), Sym("periods"))
				case 's':
					appendTokens(text, Num(f), Sym("seconds"))
				case 't':
					appendTokens(text, Num(f), Sym("ticks"))
				default:
					appendTokens(text, Num(f))
				}
			} else if noteRegex.MatchString(text) {
				note := strings.ToLower(text)
				base := map[byte]int{
					'c': 0,
					'd': 2,
					'e': 4,
					'f': 5,
					'g': 7,
					'a': 9,
					'b': 11,
				}[note[0]]
				acc := 0
				if note[1] == '#' {
					acc = 1
				}
				octave := int(note[2] - '0')
				midi := octave*12 + base + acc
				appendTokens(text, Num(midi))
			} else {
				if len(text) > 1 {
					switch text[0] {
					case '@':
						appendTokens(text, Str(text[1:]), Sym("get"))
					case '>':
						if text == ">=" {
							appendTokens(text, Sym(text))
						} else {
							appendTokens(text, Str(text[1:]), Sym("set"))
						}
					default:
						appendTokens(text, Sym(text))
					}
				} else {
					appendTokens(text, Sym(text))
				}
			}
		default:
			return nil, Err{Pos: s.Position, Err: fmt.Errorf("parse error at %s: %s", s.Position, s.TokenText())}
		}
	}
	return code, nil
}

func (vm *VM) FindMethod(name string) Fun {
	nargs := 1
	index := len(vm.valStack) - 1
	for index >= 0 && nargs < 4 {
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

func (vm *VM) Err(err error) error {
	var maybeErr Err
	if errors.As(err, &maybeErr) {
		// Preserve existing position information if already wrapped.
		return maybeErr
	}
	// Prefer the most recent non-prelude token on the stack (i.e., a user call
	// site), falling back to the innermost token that raised the error.
	var fallback *Token
	stack := vm.tokenStack.Get()
	for i := len(stack) - 1; i >= 0; i-- {
		if tok := stack[i]; tok != nil {
			if tok.pos.Filename != "<prelude>" {
				return Err{Pos: tok.pos, Err: err}
			}
			if fallback == nil {
				fallback = tok
			}
		}
	}
	if fallback != nil {
		return Err{Pos: fallback.pos, Err: err}
	}
	return Err{Err: err}
}

func (vm *VM) Errorf(format string, a ...any) error {
	return vm.Err(fmt.Errorf(format, a...))
}

var ErrEvalCancelled = errors.New("VM evaluation cancelled")

func (vm *VM) Eval(val Val) error {
	if !vm.isEvaluating.Get() {
		// someone called vm.CancelEvaluation()
		return ErrEvalCancelled
	}
	// Also honor explicit cancel signal if present.
	vm.evalMu.Lock()
	cancelCh := vm.cancelCh
	vm.evalMu.Unlock()
	if cancelCh != nil {
		select {
		case <-cancelCh:
			return ErrEvalCancelled
		default:
		}
	}
	v := val.getVal()
	if vm.IsQuoting() {
		if v == Sym("{") {
			vm.quoteDepth++
			vm.quoteBuffer = append(vm.quoteBuffer, val)
		} else if v == Sym("}") {
			if vm.quoteDepth == 0 {
				return vm.Errorf("attempt to unquote when quoteDepth == 0")
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
		err := e.Eval(vm)
		if err == nil {
			return nil
		}
		return vm.Err(err)
	}
	vm.Push(v)
	return nil
}

func (vm *VM) ParseAndEval(r io.Reader, filename string) (err error) {
	// Set up per-run cancellation + completion signals.
	vm.evalMu.Lock()
	vm.isEvaluating.Set(true)
	// Keep previous eval result until eval success, but clear any errors
	vm.errResult = nil
	vm.cancelCh = make(chan struct{})
	vm.doneCh = make(chan struct{})
	doneCh := vm.doneCh
	vm.evalMu.Unlock()

	defer func() {
		// Always mark evaluation complete and unblock any CancelEvaluation waiters.
		vm.isEvaluating.Set(false)
		vm.evalMu.Lock()
		close(doneCh)
		// Clear channels so a later CancelEvaluation doesn't wait on an old run.
		vm.cancelCh = nil
		vm.doneCh = nil
		vm.evalMu.Unlock()

		vm.Reset()
	}()

	code, parseErr := vm.Parse(r, filename)
	if parseErr != nil {
		vm.errResult = parseErr
		return parseErr
	}
	err = vm.Eval(code)
	if err != nil {
		if !errors.Is(err, ErrEvalCancelled) {
			vm.errResult = err
		}
		return err
	}
	result := vm.Top()
	if stream, ok := result.(Stream); ok {
		if stream.nframes > 0 {
			result = stream.Take(nil, stream.nframes)
		}
	}
	vm.evalResult = result
	return nil
}

func (vm *VM) ReportTapeProgress(t *Tape, nftotal, nfdone int) {
	if vm.tapeProgressCallback != nil && vm.IsEvaluating() {
		vm.tapeProgressCallback(t, nftotal, nfdone)
	}
}
