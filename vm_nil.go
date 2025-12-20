package main

type NilType struct{}

func (nil NilType) implVal() {}

var Nil = NilType{}
