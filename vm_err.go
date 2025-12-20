package main

type Err struct {
	err error
}

func (e Err) getVal() Val {
	return e
}

func (e Err) String() string {
	return e.err.Error()
}

func (e Err) Error() string {
	return e.err.Error()
}
