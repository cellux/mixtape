package main

import (
	"slices"
)

const MaxUndo = 64

type UndoFunc = func()
type UndoableFunction = func() UndoFunc

type Action struct {
	doFunc   UndoableFunction
	undoFunc UndoFunc
}

var actionList []Action

func DispatchAction(f UndoableFunction) {
	undoFunc := f()
	actionList = append(actionList, Action{f, undoFunc})
	if len(actionList) > MaxUndo {
		actionList = slices.Delete(actionList, 0, len(actionList)-MaxUndo)
	}
}

func UndoLastAction() {
	if len(actionList) == 0 {
		return
	}
	lastAction := actionList[len(actionList)-1]
	actionList = actionList[:len(actionList)-1]
	lastAction.undoFunc()
}
