package main

import (
	"reflect"
)

type Fun func(vm *VM) error

func (f Fun) Execute(vm *VM) error {
	return f(vm)
}

type Method struct {
	nargs int
	fun   Fun
}

type MethodMap map[string][]Method

func (mm MethodMap) RegisterMethod(name string, nargs int, fun Fun) {
	if _, ok := mm[name]; !ok {
		mm[name] = make([]Method, 0, 8)
	}
	mm[name] = append(mm[name], Method{nargs, fun})
}

func (mm MethodMap) FindMethod(name string, nargs int) Fun {
	if _, ok := mm[name]; !ok {
		return nil
	}
	for _, method := range mm[name] {
		if method.nargs == nargs {
			return method.fun
		}
	}
	return nil
}

type TypeMethodMap map[reflect.Type]MethodMap

var typeMethods = make(TypeMethodMap)

func RegisterMethod[T any](name string, nargs int, fun Fun) {
	t := reflect.TypeFor[T]()
	if _, ok := typeMethods[t]; !ok {
		typeMethods[t] = make(MethodMap)
	}
	typeMethods[t].RegisterMethod(name, nargs, fun)
}

func FindMethod(val Val, name string, nargs int) Fun {
	t := reflect.TypeOf(val)
	if _, ok := typeMethods[t]; !ok {
		return nil
	}
	return typeMethods[t].FindMethod(name, nargs)
}
