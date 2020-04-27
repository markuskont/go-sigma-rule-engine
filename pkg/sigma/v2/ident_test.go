package sigma

import (
	"testing"

	"gopkg.in/yaml.v2"
)

var identCase1 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
`

var identCase2 = `
---
detection:
  condition: selection1 AND selection2
  selection1:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
  selection2:
    task: "Execute a Remote Command"
`

var identCase3 = `
---
detection:
  condition: selection1
  selection1:
    winlog.event_data.ScriptBlockText:
    - " -FromBase64String"
    task: "Execute a Remote Command"
`

var identCase4 = `
---
detection:
  condition: keywords
  keywords:
  - 'wget * - http* | perl'
  - 'wget * - http* | sh'
  - 'wget * - http* | bash'
  - 'python -m SimpleHTTPServer'
`
var identCase5 = `
---
detection:
  condition: selection
  selection:
    CommandLine|endswith: '.exe -S'
    ParentImage|endswith: '\services.exe'
`

type identTestCase struct {
	IdentCount int
	IdentTypes []identType
	Rule       string
	Pos, Neg   string
}

var identCases = []identTestCase{
	{IdentCount: 1, Rule: identCase1, IdentTypes: []identType{identSelection}},
	{IdentCount: 2, Rule: identCase2, IdentTypes: []identType{identSelection, identSelection}},
	{IdentCount: 1, Rule: identCase3, IdentTypes: []identType{identSelection}},
	{IdentCount: 1, Rule: identCase4, IdentTypes: []identType{identKeyword}},
	{IdentCount: 1, Rule: identCase5, IdentTypes: []identType{identSelection}},
}

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
				j++
			}
		}
		if items != c.IdentCount {
			t.Fatalf("ident case %d defined element count %d does not match processd %d",
				i+1, c.IdentCount, items)
		}
	}
}