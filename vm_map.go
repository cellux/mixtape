package main

import (
	"fmt"
)

type Map map[Val]Val

func (m Map) getVal() Val { return m }

func (m Map) Equal(other Val) bool {
	switch rhs := other.(type) {
	case Map:
		if len(m) != len(rhs) {
			return false
		}
		for k, v := range m {
			rv, rok := rhs[k]
			if !rok {
				return false
			}
			if v != rv {
				return false
			}
		}
		return true
	default:
		return false
	}
}

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
