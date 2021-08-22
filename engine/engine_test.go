package engine

import (
	"bytes"
	"fmt"
	"path/filepath"
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

	if string(stderrBuf.String()) != "\"hello world\"\n" {
		t.Fatalf("Stderr is invalid: %q", string(stderrBuf.String()))
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

func TestCanRequireLocalFiles(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	err = e.AnchorModules(filepath.Join("testdata", "imports"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.InteractiveEval(`
		let mymod = require("mymod.js");
		if (mymod.blah != "123") { throw new Error("mymod is incorrect");}
		if (mymod.sibling.blah != "123") { throw new Error("sibling is incorrect");}
		if (mymod.grandchildren.blah != "123") { throw new Error("grandchildren is incorrect");}
		if (mymod.grandchildren.parent.blah != "123") { throw new Error("parent is incorrect");}
	`)
	if err != nil {
		t.Fatalf("Should load mymod.js without any problems, but got %v", err)
	}
}

func TestLocalModulesAreAnchored(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	err = e.AnchorModules(filepath.Join("testdata", "imports"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.InteractiveEval(`
		let mymod = require("checkForLeaks.js");
	`)
	if err == nil {
		t.Fatal("Containement leak!")
	}
}

func TestRemoteModules(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	err = e.AnchorModules(filepath.Join("testdata", "imports"))
	if err != nil {
		t.Fatal(err)
	}

	remote, done := serveRemoteModules(t, filepath.Join("testdata", "remote"), "/mods/")
	defer done()

	_, err = e.InteractiveEval(fmt.Sprintf(`
		let mymod = require("%v");
		if (mymod.msg !== "hello") {
			throw new Error("Msg should be hello but got " + mymod.msg);
		}
		if (mymod.submod.msg !== "other") {
			throw new Error("A remote module should be able to download other items");
		}
	`, fmt.Sprintf("%v/mods/valid.js", remote)))
	if err != nil {
		t.Fatalf("Should load mymod.js without any problems, but got %v", err)
	}
}

func TestBuiltinsAreProtectedFromRemote(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	err = e.AnchorModules(filepath.Join("testdata", "imports"))
	if err != nil {
		t.Fatal(err)
	}

	remote, done := serveRemoteModules(t, filepath.Join("testdata", "remote"), "/mods/")
	defer done()

	_, err = e.InteractiveEval(fmt.Sprintf(`
		let mymod = require("%v");
		if (mymod.msg !== "hello") {
			throw new Error("Msg should be hello but got " + mymod.msg);
		}
	`, fmt.Sprintf("%v/mods/invalid.js", remote)))
	if err == nil {
		t.Fatalf("A remote module should never be able to open a restricted builtin")
	}
}

func TestLocalFilesAreProtected(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	err = e.AnchorModules(filepath.Join("testdata", "imports"))
	if err != nil {
		t.Fatal(err)
	}

	remote, done := serveRemoteModules(t, filepath.Join("testdata", "remote"), "/mods/")
	defer done()

	_, err = e.InteractiveEval(fmt.Sprintf(`
		let mymod = require("%v");
		if (mymod.msg !== "hello") {
			throw new Error("Msg should be hello but got " + mymod.msg);
		}
	`, fmt.Sprintf("%v/mods/invalidLocal.js", remote)))
	if err == nil {
		t.Fatalf("A remote module should never be able to download a local fil")
	}
}
