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

func TestBasicRequire(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	_, err = e.InteractiveEval(`
		let jtb = require("@jtb");
		console.info("Version: ", jtb.version);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestModuleIsNotRestricted(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	_, err = e.InteractiveEval(`
		let rawexec = require("@rawexec");
	`)
	if err == nil {
		t.Fatal("Should never allow a restricted module to be required!")
	}
	if _, ok := e.IsRestrictedModule(err); !ok {
		t.Fatalf("Should be a restricted module but got %#v", err)
	}
	e, err = New()
	if err != nil {
		t.Fatal(err)
	}
	e.Unrestrict("@rawexec")
	_, err = e.InteractiveEval(`
		let rawexec = require("@rawexec");
	`)
	if err != nil {
		t.Fatalf("After removing restriction, module should be loadable! But got %v", err)
	}
}
