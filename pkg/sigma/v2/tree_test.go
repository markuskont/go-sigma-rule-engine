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
		p, err := NewTree(RuleHandle{Rule: rule, NoCollapseWS: c.noCollapseWSNeg})
		if err != nil {
			t.Fatal(err)
		}

		var obj DynamicMap
		// Positive cases
		for _, c := range c.Pos {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("tree parsercase %d positive case json unmarshal error %s", i+1, err)
			}
			match, _ := p.Match(obj)
			if !match {
				t.Fatalf("tree parser case %d positive case did not match", i+1)
			}
		}
		// Negative cases
		for _, c := range c.Neg {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("tree parser case %d positive case json unmarshal error %s", i+1, err)
			}
			match, _ := p.Match(obj)
			if match {
				t.Fatalf("tree parser case %d negative case matched", i+1)
			}
		}
	}
}

func benchmarkCase(b *testing.B, rawRule, rawEvent string) {
	var rule Rule
	if err := yaml.Unmarshal([]byte(parseTestCases[0].Rule), &rule); err != nil {
		b.Fail()
	}
	p, err := NewTree(RuleHandle{Rule: rule})
	if err != nil {
		b.Fail()
	}
	var event DynamicMap
	if err := json.Unmarshal([]byte(parseTestCases[0].Pos[0]), &event); err != nil {
		b.Fail()
	}
	for i := 0; i < b.N; i++ {
		p.Match(event)
	}
}

func BenchmarkTreePositive0(b *testing.B) {
	benchmarkCase(b, parseTestCases[0].Rule, parseTestCases[0].Pos[0])
}
func BenchmarkTreePositive1(b *testing.B) {
	benchmarkCase(b, parseTestCases[1].Rule, parseTestCases[1].Pos[0])
}
func BenchmarkTreePositive2(b *testing.B) {
	benchmarkCase(b, parseTestCases[2].Rule, parseTestCases[2].Pos[0])
}
func BenchmarkTreePositive3(b *testing.B) {
	benchmarkCase(b, parseTestCases[3].Rule, parseTestCases[3].Pos[0])
}
func BenchmarkTreePositive4(b *testing.B) {
	benchmarkCase(b, parseTestCases[4].Rule, parseTestCases[4].Pos[0])
}
func BenchmarkTreePositive5(b *testing.B) {
	benchmarkCase(b, parseTestCases[5].Rule, parseTestCases[6].Pos[0])
}
func BenchmarkTreePositive6(b *testing.B) {
	benchmarkCase(b, parseTestCases[6].Rule, parseTestCases[6].Pos[0])
}

func BenchmarkTreeNegative0(b *testing.B) {
	benchmarkCase(b, parseTestCases[0].Rule, parseTestCases[0].Neg[0])
}
func BenchmarkTreeNegative1(b *testing.B) {
	benchmarkCase(b, parseTestCases[1].Rule, parseTestCases[1].Neg[0])
}
func BenchmarkTreeNegative2(b *testing.B) {
	benchmarkCase(b, parseTestCases[2].Rule, parseTestCases[2].Neg[0])
}
func BenchmarkTreeNegative3(b *testing.B) {
	benchmarkCase(b, parseTestCases[3].Rule, parseTestCases[3].Neg[0])
}
func BenchmarkTreeNegative4(b *testing.B) {
	benchmarkCase(b, parseTestCases[4].Rule, parseTestCases[4].Neg[0])
}
func BenchmarkTreeNegative5(b *testing.B) {
	benchmarkCase(b, parseTestCases[5].Rule, parseTestCases[6].Neg[0])
}
func BenchmarkTreeNegative6(b *testing.B) {
	benchmarkCase(b, parseTestCases[6].Rule, parseTestCases[6].Neg[0])
}
