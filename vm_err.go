package main

import (
	"fmt"
	"text/scanner"
)

type Err struct {
	Pos scanner.Position
	Err error
}

func (e Err) getVal() Val {
	return e
}

func (e Err) String() string {
	return e.Err.Error()
}

func (e Err) Error() string {
	if e.Pos.Line == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.Pos.Filename, e.Pos.Line, e.Pos.Column, e.Err)
}

func (e Err) Unwrap() error { return e.Err }

func makeErr(err error) Err {
	if wrappedErr, ok := err.(Err); ok {
		return wrappedErr
	}
	return Err{Err: err}
}
