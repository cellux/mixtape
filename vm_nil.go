package main

type NilType struct{}

func (nil NilType) getVal() Val { return Nil }

func (nil NilType) String() string { return "nil" }

var Nil = NilType{}
