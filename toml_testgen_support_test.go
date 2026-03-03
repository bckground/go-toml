package toml_test

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/internal/testsuite"
)

func TestTOMLTest_Invalid(t *testing.T) {
	names := make([]string, 0, len(testgenInvalidCases))
	for name := range testgenInvalidCases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			input := testgenInvalidCases[name]
			t.Logf("Input TOML:\n%s", input)

			doc := map[string]interface{}{}
			err := testsuite.Unmarshal([]byte(input), &doc)

			if err == nil {
				out, err := json.Marshal(doc)
				if err != nil {
					panic("could not marshal map to json")
				}
				t.Log("JSON output from unmarshal:", string(out))
				t.Fatalf("test did not fail")
			}
		})
	}
}

func TestTOMLTest_Valid(t *testing.T) {
	names := make([]string, 0, len(testgenValidCases))
	for name := range testgenValidCases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			input := testgenValidCases[name]
			jsonRef := testgenValidJSONRef[name]
			t.Logf("Input TOML:\n%s", input)

			// TODO: change this to interface{}
			var doc map[string]interface{}

			err := testsuite.Unmarshal([]byte(input), &doc)
			if err != nil {
				de := &toml.DecodeError{}
				if errors.As(err, &de) {
					t.Logf("%s\n%s", err, de)
				}
				t.Fatalf("failed parsing toml: %s", err)
			}
			j, err := testsuite.ValueToTaggedJSON(doc)
			assert.NoError(t, err)

			var ref interface{}
			err = json.Unmarshal([]byte(jsonRef), &ref)
			assert.NoError(t, err)

			var actual interface{}
			err = json.Unmarshal(j, &actual)
			assert.NoError(t, err)

			testsuite.CmpJSON(t, "", ref, actual)
		})
	}
}
