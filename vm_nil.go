package main

type NilType struct{}

func (nil NilType) getVal() Val { return Nil }

var Nil = NilType{}
