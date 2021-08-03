package engine

import (
	"bytes"
	"testing"
)

func TestBasicRuntime(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	stderrBuf := bytes.Buffer{}
	err = e.SetStderr(&stderrBuf)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.InteractiveEval(`console.info("hello world")`)
	if err != nil {
		t.Fatalf("Unable to run code: %v", err)
	}

	if string(stderrBuf.String()) != `"hello world"` {
		t.Fatalf("Stderr is invalid: %v", string(stderrBuf.String()))
	}

	_ = e
}
