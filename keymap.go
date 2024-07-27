package main

type KeyHandler func(key string) bool

func CreateKeyHandler(f func()) KeyHandler {
	return func(key string) bool {
		f()
		return true
	}
}

type KeyMap map[string]KeyHandler

func CreateKeyMap() KeyMap {
	return KeyMap{}
}

func (km KeyMap) HandleKey(key string) bool {
	if handler, ok := km[key]; ok {
		return handler(key)
	} else {
		return false
	}
}

func (km KeyMap) Bind(key string, handler KeyHandler) {
	km[key] = handler
}
