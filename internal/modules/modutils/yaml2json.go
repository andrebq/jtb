package modutils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
	goyml "gopkg.in/yaml.v3"
)

// process all documents from input and return them encode as json
//
// if only one document exists, then it returns a single object,
// otherwise it returns a json.
//
// Keys that are not strings will be converted to strings, except
// null and binary which will not be processed.
//
// Empty documents are added ass empty json documents, eg.:
//
//
// example.yaml:
// name: "first"
// ---
// name: "second"
// ---
// ---
// 10: "int key"
// 10.1: "float key"
//
// Will be encoded as
// example.json:
// [
// 	{ "name": "fist"},
// 	{ "name": "second"},
// 	{},
// 	{ "10": "int key", "10.1": "float key"}
// ]
//
// Non-string keys are encoded using fmt.Sprintf("%v", nonStringValue),
// except for binary/null keys which are not supported at all!
//
// Binary data is encoded to base64
//
// This whole code is probably full of bugs that might cause some
// unexpected behaviour, but for that matter, using non string keys in
// Yaml comes with its own set of challenges...
//
// So I hope people won't be too mad about some inconsistencies here...
// (famous last words rigth?!)
func YamlToJSON(input string) ([]byte, error) {
	var acc []interface{}

	dec := goyml.NewDecoder(bytes.NewBufferString(input))
	for {
		var item interface{}
		err := dec.Decode(&item)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		// process the item
		if item == nil {
			// empty doc
			acc = append(acc, struct{}{})
			continue
		}
		item, err = yamlToCompatibleJSON(item)
		if err != nil {
			return nil, err
		}
		acc = append(acc, item)
	}

	if len(acc) == 0 {
		return []byte(""), nil
	} else if len(acc) == 1 {
		return json.Marshal(acc[0])
	}
	return json.Marshal(acc)
}

func yamlDocToJSONDoc(item map[string]interface{}) (map[string]interface{}, error) {
	var err error
	for k, v := range item {

		item[k], err = yamlToCompatibleJSON(v)
		if err != nil {
			return nil, err
		}

	}
	return item, nil
}

func yamlToCompatibleJSON(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case []interface{}:
		return yamlArrayToJSONArray(v)
	case map[string]interface{}:
		return yamlDocToJSONDoc(v)
	case map[interface{}]interface{}:
		return yamlToJSONLossy(v)
	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	}
	return v, nil
}

func yamlToJSONLossy(input map[interface{}]interface{}) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	var err error
	for k, v := range input {
		if k == nil {
			return nil, errors.New("cannot convert null keys from yaml to valid JSON keys")
		}
		switch k := k.(type) {
		case []byte:
			return nil, errors.New("cannot convert binary keys from json to valid JSON values")
		case string:
			ret[k], err = yamlToCompatibleJSON(v)
			if err != nil {
				return nil, err
			}
		default:
			ret[fmt.Sprintf("%v", k)], err = yamlToCompatibleJSON(v)
			if err != nil {
				return nil, err
			}
		}
	}
	return ret, nil
}

func yamlArrayToJSONArray(array []interface{}) ([]interface{}, error) {
	var err error
	for i, v := range array {
		array[i], err = yamlToCompatibleJSON(v)
		if err != nil {
			return nil, err
		}
	}
	return array, nil
}

func ToYAMLStr(in interface{}) (string, error) {
	buf := &bytes.Buffer{}
	err := yaml.NewEncoder(buf).Encode(in)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
