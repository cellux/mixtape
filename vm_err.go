package main

type Err struct {
	err error
}

func (e Err) implVal() {}

func (e Err) Error() string {
	return e.err.Error()
}
