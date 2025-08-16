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

type KeyHandler func()

// KeyMap maps keys to KeyHandlers or KeyMaps
type KeyMap map[Key]any

func CreateKeyMap(parent KeyMap) KeyMap {
	km := make(KeyMap)
	if parent != nil {
		km["_parent"] = parent
	}
	return km
}

func (km KeyMap) findOrCreateKeyMap(keyseq KeySeq, createMissing bool) KeyMap {
	current := km
	for len(keyseq) > 1 {
		k := keyseq[0]
		if keymap, ok := current[k].(KeyMap); ok {
			current = keymap
		} else if createMissing {
			keymap := make(KeyMap)
			current[k] = keymap
			current = keymap
		} else {
			return nil
		}
		keyseq = keyseq[1:]
	}
	return current
}

func (km KeyMap) findKeyMap(keyseq KeySeq) KeyMap {
	return km.findOrCreateKeyMap(keyseq, false)
}

func (km KeyMap) HandleKeySeq(keyseq KeySeq) KeyMap {
	keymap := km.findKeyMap(keyseq)
	if keymap != nil {
		k := keyseq[len(keyseq)-1]
		if value, ok := keymap[k]; ok {
			switch v := value.(type) {
			case KeyHandler:
				v()
				return nil
			case KeyMap:
				return v
			}
		}
	}
	if parent, ok := km["_parent"]; ok {
		return parent.(KeyMap).HandleKeySeq(keyseq)
	}
	return nil
}

func (km KeyMap) Bind(keys string, handler KeyHandler) {
	keyseq := strings.Fields(keys)
	keymap := km.findOrCreateKeyMap(keyseq, true)
	k := keyseq[len(keyseq)-1]
	keymap[k] = handler
}

type KeyMapManager struct {
	currentMap     KeyMap
	currentKeys    KeySeq
	isResetPending bool
}

func CreateKeyMapManager() *KeyMapManager {
	keymap := CreateKeyMap(nil)
	return &KeyMapManager{currentMap: keymap}
}

func (kmm *KeyMapManager) IsInsideKeySequence() bool {
	return kmm.currentKeys != nil
}

func (kmm *KeyMapManager) Reset() {
	kmm.currentKeys = nil
	kmm.isResetPending = false
}

func (kmm *KeyMapManager) SetCurrentKeyMap(keymap KeyMap) {
	kmm.currentMap = keymap
	kmm.Reset()
}

func (kmm *KeyMapManager) HandleKey(key Key) {
	if kmm.isResetPending {
		kmm.Reset()
	}
	keyseq := append(kmm.currentKeys, key)
	keymap := kmm.currentMap.HandleKeySeq(keyseq)
	if keymap != nil {
		kmm.currentKeys = keyseq
	} else {
		kmm.isResetPending = true
	}
}
