package sigma

import (
	"encoding/json"
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
)

type identExampleType int

const (
	identNA identExampleType = iota
	ident1
)

type simpleKeywordAuditEventExample1 struct {
	Command string `json:"cmd"`
}

// Keywords implements Keyworder
func (s simpleKeywordAuditEventExample1) Keywords() ([]string, bool) {
	return []string{s.Command}, true
}

// Select implements Selector
func (s simpleKeywordAuditEventExample1) Select(_ string) (interface{}, bool) {
	return nil, false
}

var identSelection1 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
`

var identSelection2 = `
---
detection:
  condition: selection1 AND selection2
  selection1:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
  selection2:
    task: "Execute a Remote Command"
`

var identSelection3 = `
---
detection:
  condition: selection1
  selection1:
    winlog.event_data.ScriptBlockText:
    - " -FromBase64String"
    task: "Execute a Remote Command"
`

var identSelection4 = `
---
detection:
  condition: selection
  selection:
    CommandLine|endswith: '.exe -S'
    ParentImage|endswith: '\services.exe'
`

var identKeyword1 = `
---
detection:
  condition: keywords
  keywords:
  - 'bash -c'
  - 'cat /etc/shadow'
`

var identKeyword1pos1 = `
{ "cmd": "sudo bash -c \"cat /etc/shadow /etc/group /etc/passwd\"" }
`
var identKeyword1neg1 = `
{ "cmd": "sh -c \"cat /etc/resolv.conf\"" }
`
var identKeyword2 = `
---
detection:
  condition: keywords
  keywords:
  - 'wget * - http* | perl'
  - 'wget * - http* | sh'
  - 'wget * - http* | bash'
  - "*python -m Simple*Server"
`

var identKeyword2pos1 = `
{ "cmd": "/usr/bin/python -m SimpleHTTPServer" }
`
var identKeyword2neg1 = `
{ "cmd": "/usr/bin/python -m pip install --user pip" }
`

var identKeyword3 = `
---
detection:
  condition: keywords
  keywords: "*python* -m SimpleHTTPServer"
`

type identPosNegCase struct {
	Pos, Neg Event
}

type identTestCase struct {
	IdentCount int
	IdentTypes []identType
	Rule       string
	Pos, Neg   string

	Example identExampleType
}

func (i identTestCase) sigma() (*identPosNegCase, error) {
	switch i.Example {
	case ident1:
		var pos, neg simpleKeywordAuditEventExample1
		if err := json.Unmarshal([]byte(i.Pos), &pos); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(i.Neg), &neg); err != nil {
			return nil, err
		}
		return &identPosNegCase{Pos: pos, Neg: neg}, nil
	}
	return nil, fmt.Errorf("Unknown identifier test case")
}

var selectionCases = []*identTestCase{
	{IdentCount: 1, Rule: identSelection1, IdentTypes: []identType{identSelection}},
	{IdentCount: 2, Rule: identSelection2, IdentTypes: []identType{identSelection, identSelection}},
	{IdentCount: 1, Rule: identSelection3, IdentTypes: []identType{identSelection}},
	{IdentCount: 1, Rule: identSelection4, IdentTypes: []identType{identSelection}},
}

var keywordCases = []identTestCase{
	{
		IdentCount: 1,
		Rule:       identKeyword1,
		IdentTypes: []identType{identKeyword},
		Pos:        identKeyword1pos1,
		Neg:        identKeyword1neg1,
		Example:    ident1,
	},
	{
		IdentCount: 1,
		Rule:       identKeyword2,
		IdentTypes: []identType{identKeyword},
		Pos:        identKeyword2pos1,
		Neg:        identKeyword2neg1,
		Example:    ident1,
	},
}

var identCases = keywordCases

func TestParseIdent(t *testing.T) {
	for i, c := range identCases {
		var r Rule
		if err := yaml.Unmarshal([]byte(c.Rule), &r); err != nil {
			t.Fatalf("ident case %d yaml parse fail: %s", i+1, err)
		}
		condition, ok := r.Detection["condition"].(string)
		if !ok {
			t.Fatalf("ident case %d missing condition", i+1)
		}
		l := lex(condition)
		var items, j int
		keywords := make([]Matcher, 0)
		for item := range l.items {
			switch item.T {
			case TokIdentifier:
				val, ok := r.Detection[item.Val]
				if !ok {
					t.Fatalf("ident case %d missing ident %s or unable to extract", i+1, item.Val)
				}
				items++
				if k := checkIdentType(item, val); k != c.IdentTypes[j] {
					t.Fatalf("ident case %d ident %d kind mismatch expected %s got %s",
						i+1, j+1, c.IdentTypes[j], k)
				}
				switch c.IdentTypes[j] {
				case identKeyword:
					kw, err := newKeyword(val)
					if err != nil {
						t.Fatalf("ident case %d token %d failed to parse as keyword: %s",
							i+1, j+1, err)
					}
					keywords = append(keywords, kw)
				case identSelection:

				}
				j++
			}
		}
		if items != c.IdentCount {
			t.Fatalf("ident case %d defined element count %d does not match processd %d",
				i+1, c.IdentCount, items)
		}
		cases, err := c.sigma()
		if err != nil {
			t.Fatalf("ident case %d unable to cast test cases to sigma events, err: %s",
				i+1, err)
		}
		for _, rule := range keywords {
			if rule == nil {
				t.Fatalf("ident case %d nil rule pointer", i+1)
			}
			if !rule.Match(cases.Pos) {
				t.Fatalf("ident case %d positive test case did not match %s", i+1, cases.Pos)
			}
			if rule.Match(cases.Neg) {
				t.Fatalf("ident case %d negative test case matched %s", i+1, cases.Neg)
			}
		}

		fmt.Print(cases)
	}
}
