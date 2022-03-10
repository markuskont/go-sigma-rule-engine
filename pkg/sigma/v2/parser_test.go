package sigma

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v2"
)

var detection1 = `
detection:
  condition: "selection1 and not selection3"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  selection3:
    CommandLine: "+R +H +S +A *.cui"
`

var detection1_positive = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +A asd.cui",
	"ParentImage": "C:\\test\\wmiprvse.exe",
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "aaa",
	"ParentImage": "C:\\test\\wmiprvse.exe"
}
`

var detection1_negative1 = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "C:\\test\\mshta.exe"
}
`
var detection1_negative2 = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui"
}
`
var detection2 = `
detection:
  condition: "(selection1 and selection2) and not selection3"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection2:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  selection3:
    CommandLine: "+R +H +S +A *.cui"
`

var detection3 = `
detection:
  condition: "(selection1 or selection2) and not selection3"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection2:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  selection3:
    CommandLine: "+R +H +S +A *.cui"
`

var detection3_positive1 = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +A asd.cui",
	"ParentImage": "C:\\test\\custom.exe",
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "aaa",
	"ParentImage": "C:\\test\\wmiprvse.exe"
}
`
var detection3_positive2 = `
{
	"Image":       "C:\\test\\custom.exe",
	"CommandLine": "+R +H +A asd.cui",
	"ParentImage": "C:\\test\\wmiprvse.exe",
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "aaa",
	"ParentImage": "C:\\test\\wmiprvse.exe"
}
`

var detection3_negative = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "C:\\test\\mshta.exe"
}
`

var detection4 = `
detection:
  condition: "all of selection* and not filter"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection2:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  filter:
    CommandLine: "+R +H +S +A *.cui"
`

var detection5 = `
detection:
  condition: "1 of selection* and not filter"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection2:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  filter:
    CommandLine: "+R +H +S +A *.cui"
`

var detection6 = `
detection:
  condition: "all of them"
  selection_images:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection_parent_images:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
`

var detection6_positive = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +A asd.cui",
	"ParentImage": "C:\\test\\wmiprvse.exe",
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "aaa",
	"ParentImage": "C:\\test\\wmiprvse.exe"
}
`

var detection6_negative = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "C:\\test\\mshta\\lll.exe"
}
`

var detection7 = `
detection:
  condition: "1 of them"
  selection_images:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
  selection_parent_images:
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
`

var detection7_negative1 = `
{
	"Image":       "C:\\test\\bytesadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "E:\\go\\bin\\gofmt"
}
`
var detection7_negative2 = `
{
	"Image":       "C:\\test\\bytesadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui"
}
`

var detection8 = `
detection:
  condition: "selection1 and not selection3"
  selection1:
    Image:
    - '*\schtasks.exe'
    - '*\nslookup.exe'
    - '*\certutil.exe'
    - '*\bitsadmin.exe'
    - '*\mshta.exe'
    ParentImage:
    - '*\mshta.exe'
    - '*\powershell.exe'
    - '*\cmd.exe'
    - '*\rundll32.exe'
    - '*\cscript.exe'
    - '*\wscript.exe'
    - '*\wmiprvse.exe'
  selection3:
    CommandLine: "+R +H +S +A *.cui"
`

var detection8_positive = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +A asd.cui",
	"ParentImage": "C:\\test\\wmiprvse.exe",
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "aaa",
	"ParentImage": "C:\\test\\wmiprvse.exe"
}
`

var detection8_negative1 = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "C:\\test\\mshta.exe"
}
`
var detection8_negative2 = `
{
	"Image":       "C:\\test\\bitsadmin.exe",
	"ParentImage": "C:\\test\\mshta.exe"
}
`

var detection9 = `
detection:
  condition: "selection"
  selection:
    - PipeName|re: '\\\\SomePipeName[0-9a-f]{2}'
    - PipeName2|re: '\\\\AnotherPipe[0-9a-f]*Name'
`

var detection9_positive = `
{
	"PipeName":       "\\\\SomePipeNamea4",
	"PipeName2":       "\\\\AnotherPipe0af3Name"
}
`

var detection9_negative = `
{
	"PipeName":       "\\\\SomePipeNameZZ",
	"PipeName2":       "\\\\AnotherPipe01zzName"
}
`

var detection10 = `
detection:
  condition: "selection1 and selection2"
  selection1:
    - SomeName|startswith: 'TestStart'
  selection2:
    - SomeName|endswith: 'TestEnd'
`

var detection10_positive = `
{
	"SomeName":       "TestStart-Value-TestEnd"
}
`

var detection10_negative = `
{
	"SomeName":       "TestStart-Value"
}
`

var detection11 = `
detection:
  condition: "selection1 and selection2"
  selection1:
    SomeName|contains|all: 
      - 'mark1'
      - 'mark2'
  selection2:
    SomeName|contains:
      - 'version1'
      - 'version2'
`

var detection11_positive = `
{
	"SomeName":       "Some mark1 mark2 String version2"
}
`

var detection11_negative = `
{
	"SomeName":       "mark1 mark2 mark3 non-matching string"
}
`

var detection12 = `
detection:
  condition: "selection1 and selection2"
  selection1:
    SomeKey|contains|all:
      - 'val1'
      - 'val2'
  selection2:
    SomeKey2:
      - 'mustMatch1'
      - 'mustMatch2'
`

var detection12_positive = `
{
	"SomeKey":       "val1 val2",
	"SomeKey2":      "mustMatch1"
}
`

var detection12_negative = `
{
	"SomeKey":       "val1 val2",
	"SomeKey2":      "mustMatch3"
}
`

type parseTestCase struct {
	Rule     string
	Pos, Neg []string
}

var parseTestCases = []parseTestCase{
	{
		Rule: detection1,
		Pos:  []string{detection1_positive},
		Neg:  []string{detection1_negative1, detection1_negative2},
	},
	{
		Rule: detection2,
		Pos:  []string{detection1_positive},
		Neg:  []string{detection1_negative1, detection1_negative2},
	},
	{
		Rule: detection3,
		Pos:  []string{detection3_positive1, detection3_positive2},
		Neg:  []string{detection3_negative},
	},
	{
		Rule: detection4,
		Pos:  []string{detection1_positive},
		Neg:  []string{detection1_negative1, detection1_negative2},
	},
	{
		Rule: detection5,
		Pos:  []string{detection3_positive1, detection3_positive2},
		Neg:  []string{detection3_negative},
	},
	{
		Rule: detection6,
		Pos:  []string{detection6_positive},
		Neg:  []string{detection6_negative},
	},
	{
		Rule: detection7,
		Pos:  []string{detection3_positive1, detection3_positive2},
		Neg:  []string{detection7_negative1, detection7_negative2},
	},
	{
		Rule: detection8,
		Pos:  []string{detection8_positive},
		Neg:  []string{detection8_negative1, detection8_negative2},
	},
	{
		Rule: detection9,
		Pos:  []string{detection9_positive},
		Neg:  []string{detection9_negative},
	},
	{
		Rule: detection10,
		Pos:  []string{detection10_positive},
		Neg:  []string{detection10_negative},
	},
	{
		Rule: detection11,
		Pos:  []string{detection11_positive},
		Neg:  []string{detection11_negative},
	},
	{
		Rule: detection12,
		Pos:  []string{detection12_positive},
		Neg:  []string{detection12_negative},
	},
}

func TestTokenCollect(t *testing.T) {
	for _, c := range LexPosCases {
		p := &parser{
			lex: lex(c.Expr),
		}
		if err := p.collect(); err != nil {
			switch err.(type) {
			case ErrUnsupportedToken:
			default:
				t.Fatal(err)
			}
		}
	}
}

func TestParse(t *testing.T) {
	for i, c := range parseTestCases {
		var rule Rule
		if err := yaml.Unmarshal([]byte(c.Rule), &rule); err != nil {
			t.Fatalf("rule parse case %d failed to unmarshal yaml, %s", i+1, err)
		}
		expr := rule.Detection["condition"].(string)
		p := &parser{
			lex:   lex(expr),
			sigma: rule.Detection,
		}
		if err := p.collect(); err != nil {
			t.Fatalf("rule parser case %d failed to collect lexical tokens, %s", i+1, err)
		}
		if err := p.parse(); err != nil {
			switch err.(type) {
			case ErrWip:
				t.Fatalf("WIP")
			default:
				t.Fatalf("rule parser case %d failed to parse lexical tokens, %s", i+1, err)
			}
		}
		var obj DynamicMap
		// Positive cases
		for _, c := range c.Pos {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("rule parser case %d positive case json unmarshal error %s", i+1, err)
			}
			m, _ := p.result.Match(obj)
			if !m {
				t.Fatalf("rule parser case %d positive case did not match", i+1)
			}
		}
		// Negative cases
		for _, c := range c.Neg {
			if err := json.Unmarshal([]byte(c), &obj); err != nil {
				t.Fatalf("rule parser case %d positive case json unmarshal error %s", i+1, err)
			}
			m, _ := p.result.Match(obj)
			if m {
				t.Fatalf("rule parser case %d negative case matched", i+1)
			}
		}
	}
}
