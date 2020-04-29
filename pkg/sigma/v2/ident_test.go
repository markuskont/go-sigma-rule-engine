package sigma

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func getField(key string, data map[string]interface{}) (interface{}, bool) {
	if data == nil {
		return nil, false
	}
	bits := strings.SplitN(key, ".", 2)
	if len(bits) == 0 {
		return nil, false
	}
	if val, ok := data[bits[0]]; ok {
		switch res := val.(type) {
		case map[string]interface{}:
			return getField(bits[1], res)
		default:
			return val, ok
		}
	}
	return nil, false
}

type identExampleType int

const (
	identNA identExampleType = iota
	ident1
	ident2
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

type simpleDynamicMapEventExample1 map[string]interface{}

// Keywords implements Keyworder
func (s simpleDynamicMapEventExample1) Keywords() ([]string, bool) {
	return nil, false
}

// Select implements Selector
func (s simpleDynamicMapEventExample1) Select(key string) (interface{}, bool) {
	return getField(key, s)
}

var identSelection1 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
    - '::FromBase64String'
`

var identSelection1pos1 = `
{
  "event_id": 4104,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
  "event_data": {
    "MessageNumber": "1",
    "MessageTotal": "1",
		"ScriptBlockText": "$s=New-Object IO.MemoryStream(,[Convert]::FromBase64String(\"OMITTED BASE64 STRING\"));",
    "ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
  }
}
`

var identSelection2 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
    - '::FromBase64String'
    task: 
    - 'Remote Command'
`

var identSelection3 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
    - '::FromBase64String'
    task: 
    - 'Remote Command'
    channel|endswith: "PowerShell/Operational"
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
  keywords:
  - '/\S+python.* -m Simple\w+Server.*/'
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
	case ident2:
		var pos, neg simpleDynamicMapEventExample1
		if i.Pos != "" {
			if err := json.Unmarshal([]byte(i.Pos), &pos); err != nil {
				return nil, err
			}
		}
		if i.Neg != "" {
			if err := json.Unmarshal([]byte(i.Neg), &neg); err != nil {
				return nil, err
			}
		}
		return &identPosNegCase{Pos: pos, Neg: neg}, nil
	}
	return nil, fmt.Errorf("Unknown identifier test case")
}

var selectionCases = []identTestCase{
	{
		IdentCount: 1,
		Rule:       identSelection1,
		IdentTypes: []identType{identSelection},
		Pos:        identSelection1pos1,
		Example:    ident2,
	},
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
	{
		IdentCount: 1,
		Rule:       identKeyword3,
		IdentTypes: []identType{identKeyword},
		Pos:        identKeyword2pos1,
		Neg:        identKeyword2neg1,
		Example:    ident1,
	},
}

var identCases = append(keywordCases, selectionCases...)

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
					kw, err := NewKeyword(val)
					if err != nil {
						t.Fatalf("ident case %d token %d failed to parse as keyword: %s",
							i+1, j+1, err)
					}
					keywords = append(keywords, kw)
				case identSelection:
					_, err := NewSelection(val)
					if err != nil {
						t.Fatalf("ident case %d token %d failed to parse as selection: %s",
							i+1, j+1, err)
					}
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
			if c.Pos != "" && !rule.Match(cases.Pos) {
				t.Fatalf("ident case %d positive test case did not match %s", i+1, cases.Pos)
			}
			if c.Neg != "" && rule.Match(cases.Neg) {
				t.Fatalf("ident case %d negative test case matched %s", i+1, cases.Neg)
			}
		}
	}
}
