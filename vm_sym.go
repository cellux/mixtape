package main

type Sym string

func (s Sym) String() string {
	return string(s)
}
