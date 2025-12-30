package main

import (
	"strings"
)

// Key is a string representation of a key or key chord.
//
// Examples: "f" "C-c" "C-M-f"
//
// Modifier prefixes should be in the following order: C-M-S-
type Key = string

type KeySeq = []Key

type KeyHandler interface {
	HandleKey(key Key) (nextHandler KeyHandler, handled bool)
}

// KeyMap maps keys to KeyHandlers or KeyMaps
type KeyMap map[Key]any

func CreateKeyMap() KeyMap {
	return make(KeyMap)
}

func (km KeyMap) BindSeq(keyseq KeySeq, handler any) {
	if len(keyseq) == 0 {
		return
	}
	k := keyseq[0]
	if len(keyseq) == 1 {
		km[k] = handler
		return
	}
	value, exists := km[k]
	if !exists {
		value = make(KeyMap)
		km[k] = value
	}
	if _, ok := value.(KeyMap); !ok {
		value = make(KeyMap)
		km[k] = value
	}
	keymap := value.(KeyMap)
	keymap.BindSeq(keyseq[1:], handler)
}

func (km KeyMap) Bind(keys string, handler any) {
	keyseq := strings.Fields(keys)
	km.BindSeq(keyseq, handler)
}

func (km KeyMap) HandleKey(key Key) (KeyHandler, bool) {
	if v, ok := km[key]; ok {
		switch vv := v.(type) {
		case KeyMap:
			return vv, true
		case func():
			vv()
			return nil, true
		case KeyHandler:
			return vv.HandleKey(key)
		default:
			return nil, false
		}
	}
	return nil, false
}
