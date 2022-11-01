package sigma

import (
	"encoding/json"
	"testing"

	"github.com/markuskont/datamodels"
	"gopkg.in/yaml.v2"
)

func TestTreeParse(t *testing.T) {
	for _, c := range parseTestCases {
		var rule Rule
		if err := yaml.Unmarshal([]byte(c.Rule), &rule); err != nil {
			t.Fatalf("tree parse case %d failed to unmarshal yaml, %s", c.ID, err)
		}
		p, err := NewTree(RuleHandle{Rule: rule, NoCollapseWS: c.noCollapseWSNeg})
		if err != nil {
			t.Fatalf("tree parse case %d failed: %s", c.ID, err)
		}

		var obj datamodels.Map
		// Positive cases
		for i, c2 := range c.Pos {
			if err := json.Unmarshal([]byte(c2), &obj); err != nil {
				t.Fatalf("rule parser case %d positive case %d json unmarshal error %s", c.ID, i, err)
			}
			m, _ := p.Match(obj)
			if !m {
				t.Fatalf("rule parser case %d positive case %d did not match", c.ID, i)
			}
		}
		// Negative cases
		for i, c2 := range c.Neg {
			if err := json.Unmarshal([]byte(c2), &obj); err != nil {
				t.Fatalf("rule parser case %d positive case %d json unmarshal error %s", c.ID, i, err)
			}
			m, _ := p.Match(obj)
			if m {
				t.Fatalf("rule parser case %d negative case %d matched", c.ID, i)
			}
		}
	}
}

// we should probably add an alternative to this benchmark to include noCollapseWS on or off (we collapse by default now)
func benchmarkCase(b *testing.B, rawRule, rawEvent string) {
	var rule Rule
	if err := yaml.Unmarshal([]byte(parseTestCases[0].Rule), &rule); err != nil {
		b.Fail()
	}
	p, err := NewTree(RuleHandle{Rule: rule})
	if err != nil {
		b.Fail()
	}
	var event datamodels.Map
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
