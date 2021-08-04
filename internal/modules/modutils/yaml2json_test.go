package modutils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestYAML2JSON(t *testing.T) {
	// at least make sure the example works as expected...
	yaml := `
name: "first"
---
name: "second"
---
---
key:
  10: "int key"
  10.1: "float key"
`
	var expectedObject []interface{}
	err := json.Unmarshal([]byte(`
	[
		{ "name": "first"},
		{ "name": "second"},
		{},
		{ "key": { "10": "int key", "10.1": "float key"} }
	]`), &expectedObject)
	if err != nil {
		t.Fatal(err)
	}
	actualJSON, err := YamlToJSON(yaml)
	if err != nil {
		t.Fatal(err)
	}
	var actualObject []interface{}
	err = json.Unmarshal(actualJSON, &actualObject)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedObject, actualObject) {
		t.Fatalf("JSON objects do not match, got (\n%v\n) expecting (\n%v\n)", actualObject, expectedObject)
	}

}
