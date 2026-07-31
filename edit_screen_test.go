package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestEditScreenReplaysLastSuccessfulResultAfterEvaluationError(t *testing.T) {
	vm, err := CreateVM()
	if err != nil {
		t.Fatal(err)
	}

	script := []byte("1")
	if err := vm.ParseAndEval(bytes.NewReader(script), "<test>"); err != nil {
		t.Fatal(err)
	}

	es := &EditScreen{}
	es.cacheSuccessfulEvaluation(script, vm.evalResult)

	if err := vm.ParseAndEval(strings.NewReader("fqeoiqow"), "<test>"); err == nil {
		t.Fatal("expected invalid script to fail")
	}
	if vm.evalResult != nil {
		t.Fatal("failed evaluation should clear the VM's transient result")
	}
	if !es.canReplay(script) {
		t.Fatal("last successful result should remain replayable after an evaluation error")
	}
}
