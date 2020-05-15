package sigma

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestTreeParse(t *testing.T) {
	for i, c := range parseTestCases {
		var rule Rule
		if err := yaml.Unmarshal([]byte(c.Rule), &rule); err != nil {
			t.Fatalf("tree parse case %d failed to unmarshal yaml, %s", i+1, err)
		}
		p, err := NewTree(RuleHandle{Rule: rule})
		if err != nil {
			t.Fatal(err)
		}

		var obj DynamicMap
		// Positive cases
		for _, c := range c.Pos {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("tree parsercase %d positive case json unmarshal error %s", i+1, err)
			}
			if !p.Match(obj) {
				t.Fatalf("tree parser case %d positive case did not match", i+1)
			}
		}
		// Negative cases
		for _, c := range c.Neg {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("tree parser case %d positive case json unmarshal error %s", i+1, err)
			}
			if p.Match(obj) {
				t.Fatalf("tree parser case %d negative case matched", i+1)
			}
		}
	}
}
