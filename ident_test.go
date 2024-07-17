package sigma

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/markuskont/datamodels"
	"gopkg.in/yaml.v2"
)

type identExampleType int

const (
	ident1 identExampleType = iota
	ident2
)

type identPosNegCases struct {
	Pos, Neg []Event
}

type identTestCase struct {
	IdentCount int
	IdentTypes []identType
	Rule       string
	Pos, Neg   []string

	Example identExampleType
}

func (i identTestCase) sigma() (*identPosNegCases, error) {
	posContainer := make([]Event, 0)
	negContainer := make([]Event, 0)
	switch i.Example {
	case ident1:
		if i.Pos == nil || len(i.Pos) == 0 {
			return nil, fmt.Errorf("missing positive test cases")
		}
		for _, c := range i.Pos {
			var obj simpleKeywordAuditEventExample1
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				return nil, err
			}
			posContainer = append(posContainer, obj)
		}
		for _, c := range i.Neg {
			var obj simpleKeywordAuditEventExample1
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				return nil, err
			}
			negContainer = append(negContainer, obj)
		}
		return &identPosNegCases{Pos: posContainer, Neg: negContainer}, nil
	case ident2:
		if i.Pos == nil || len(i.Pos) == 0 {
			return nil, fmt.Errorf("missing positive test cases")
		}
		for _, c := range i.Pos {
			var obj datamodels.Map
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				return nil, err
			}
			posContainer = append(posContainer, obj)
		}
		if i.Neg == nil || len(i.Neg) == 0 {
			return nil, fmt.Errorf("missing negative test cases")
		}
		for _, c := range i.Neg {
			var obj datamodels.Map
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				return nil, err
			}
			negContainer = append(negContainer, obj)
		}
		return &identPosNegCases{Pos: posContainer, Neg: negContainer}, nil
	}
	return nil, fmt.Errorf("Unknown identifier test case")
}

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
    winlog.event_data.ScriptBlockText|contains:
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
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "$s=New-Object IO.MemoryStream(,[Convert]::FromBase64String(\"OMITTED BASE64 STRING\"));",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

// another command
var identSelection1neg1 = `
{
  "event_id": 4104,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "Some awesome command",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

// missing field
var identSelection1neg2 = `
{
  "event_id": 4104,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
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

var identSelection2 = `
---
detection:
  condition: selection
  selection:
    event_id:
    - 8888
    - 1337
    - 13
`

var identSelection3 = `
---
detection:
  condition: selection
  selection:
    event_id: 1337
`

var identSelection2pos1 = `
{
  "event_id": 1337,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "Some awesome command",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

var identSelection2neg1 = `
{
  "event_id": 4104,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "Some awesome command",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

var identSelection2neg2 = `
{
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "Some awesome command",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`
var identSelection4 = `
---
detection:
  condition: selection
  selection:
   - winlog.event_data.ScriptBlockText|contains:
      - '*wmic*shadowcopy*delete'
`

var identSelection4pos1 = `
{
  "event_id": 1337,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "someData WMic shaDOWcOpY     dEleTe",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

var identSelection4neg1 = `
{
  "event_id": 1337,
  "channel": "Microsoft-Windows-PowerShell/Operational",
  "task": "Execute a Remote Command",
  "opcode": "On create calls",
  "version": 1,
  "record_id": 1559,
	"winlog": {
		"event_data": {
			"MessageNumber": "1",
			"MessageTotal": "1",
			"ScriptBlockText": "something normal",
			"ScriptBlockId": "ecbb39e8-1896-41be-b1db-9a33ed76314b"
		}
	}
}
`

var selectionCases = []identTestCase{
	{
		IdentCount: 1,
		Rule:       identSelection1,
		IdentTypes: []identType{identSelection},
		Pos:        []string{identSelection1pos1},
		Neg:        []string{identSelection1neg1, identSelection1neg2},
		Example:    ident2,
	},
	{
		IdentCount: 1,
		Rule:       identSelection2,
		IdentTypes: []identType{identSelection},
		Pos:        []string{identSelection2pos1},
		Neg:        []string{identSelection2neg1, identSelection2neg2},
		Example:    ident2,
	},
	{
		IdentCount: 1,
		Rule:       identSelection3,
		IdentTypes: []identType{identSelection},
		Pos:        []string{identSelection2pos1},
		Neg:        []string{identSelection2neg1, identSelection2neg2},
		Example:    ident2,
	},
	{
		IdentCount: 1,
		Rule:       identSelection4,
		IdentTypes: []identType{identSelection},
		Pos:        []string{identSelection4pos1},
		Neg:        []string{identSelection4neg1},
		Example:    ident2,
	},
}

var keywordCases = []identTestCase{
	{
		IdentCount: 1,
		Rule:       identKeyword1,
		IdentTypes: []identType{identKeyword},
		Pos:        []string{identKeyword1pos1},
		Neg:        []string{identKeyword1neg1},
		Example:    ident1,
	},
	{
		IdentCount: 1,
		Rule:       identKeyword2,
		IdentTypes: []identType{identKeyword},
		Pos:        []string{identKeyword2pos1},
		Neg:        []string{identKeyword2neg1},
		Example:    ident1,
	},
	{
		IdentCount: 1,
		Rule:       identKeyword3,
		IdentTypes: []identType{identKeyword},
		Pos:        []string{identKeyword2pos1},
		Neg:        []string{identKeyword2neg1},
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
		selections := make([]Matcher, 0)
		for item := range l.items {
			switch item.T {
			case TokIdentifier:
				val, ok := r.Detection[item.Val]
				if !ok {
					t.Fatalf("ident case %d missing ident %s or unable to extract", i+1, item.Val)
				}
				items++
				if k := checkIdentType(item.Val, val); k != c.IdentTypes[j] {
					t.Fatalf("ident case %d ident %d kind mismatch expected %s got %s",
						i+1, j+1, c.IdentTypes[j], k)
				}
				switch c.IdentTypes[j] {
				case identKeyword:
					kw, err := NewKeyword(val, false)
					if err != nil {
						t.Fatalf("ident case %d token %d failed to parse as keyword: %s",
							i+1, j+1, err)
					}
					keywords = append(keywords, kw)
				case identSelection:
					sel, err := NewSelectionBranch(val, false)
					if err != nil {
						t.Fatalf("ident case %d token %d failed to parse as selection: %s",
							i+1, j+1, err)
					}
					selections = append(selections, sel)
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
			for j, c := range cases.Pos {
				m, _ := rule.Match(c)
				if !m {
					t.Fatalf("ident case %d positive test case %d did not match %s",
						i+1, j+1, c)
				}
			}
			for j, c := range cases.Neg {
				m, _ := rule.Match(c)
				if m {
					t.Fatalf("ident case %d negative test case %d did not match %s",
						i+1, j+1, c)
				}
			}
		}
		for _, rule := range selections {
			if rule == nil {
				t.Fatalf("ident case %d nil rule pointer", i+1)
			}
			for j, c := range cases.Pos {
				m, _ := rule.Match(c)
				if !m {
					t.Fatalf("ident case %d positive test case %d did not match %s",
						i+1, j+1, c)
				}
			}
			for j, c := range cases.Neg {
				m, _ := rule.Match(c)
				if m {
					t.Fatalf("ident case %d negative test case %d did not match %s",
						i+1, j+1, c)
				}
			}
		}
	}
}
