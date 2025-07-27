package main

import (
	"fmt"
)

type Map map[Val]Val

func (m Map) String() string {
	return fmt.Sprintf("%v", map[Val]Val(m))
}

func (m Map) GetVal(k any) Val {
	key := AsVal(k)
	return m[key]
}

func (m Map) SetVal(k, v any) {
	key := AsVal(k)
	val := AsVal(v)
	m[key] = val
}
