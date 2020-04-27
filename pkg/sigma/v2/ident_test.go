package sigma

import (
	"fmt"
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
	Rule       string
	Pos, Neg   string
}

var identCases = []identTestCase{
	{IdentCount: 1, Rule: identCase1},
	{IdentCount: 2, Rule: identCase2},
	{IdentCount: 1, Rule: identCase3},
	{IdentCount: 1, Rule: identCase4},
	{IdentCount: 1, Rule: identCase5},
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
		for item := range l.items {
			switch item.T {
			case TokIdentifier:
				val, ok := r.Detection[item.Val]
				if !ok {
					t.Fatalf("ident case %d missing ident %s or unable to extract", i+0, item.Val)
				}
				fmt.Println(val)
			}
		}
	}
}
