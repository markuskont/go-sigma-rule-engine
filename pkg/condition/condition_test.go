package condition

import (
	"testing"
)

type dummyObject map[string]string

// GetMessage implements MessageGetter
func (d dummyObject) GetMessage() []string {
	keys := []string{
		"Image",
		"CommandLine",
		"ParentImage",
	}
	res := make([]string, 0)
	for _, k := range keys {
		if val, ok := d[k]; ok {
			res = append(res, val)
		}
	}
	return res
}

// GetField returns a success status and arbitrary field content if requested map key is present
func (d dummyObject) GetField(key string) (interface{}, bool) {
	if val, ok := d[key]; ok {
		return val, ok
	}
	return nil, false
}

var data = []string{
	"aaa",
	"bbb1",
	"cccc*",
	"1 of fields*",
	"all of them",
	"aaa or bbb",
	"aaa and bbb",
	"aaa and (bbb or ccc)",
	`selection | count(dns_query) by parent_domain > 1000`,
	`( selection1 and selection2 ) or selection3`,
	`selection and ( sourceRDP or destinationRDP )`,
	`(rundll_image or rundll_ofn) and selection`,
	`(selection1 and not 1 of filter*) or selection2 or selection3 or selection4`,
}

/*
func TestLex(t *testing.T) {
	for _, str := range data {
		l := lex(str)
		out := make([]Item, 0)
		for tok := range l.items {
			out = append(out, tok)
		}
		fmt.Printf("%+v\n", out)
	}
}
*/

var detection1 = map[string]interface{}{
	"condition": "selection1 and not selection3",
	"selection1": map[string]interface{}{
		"Image": []string{
			`*\schtasks.exe`,
			`*\nslookup.exe`,
			`*\certutil.exe`,
			`*\bitsadmin.exe`,
			`*\mshta.exe`,
		},
		"ParentImage": []string{
			`*\mshta.exe`,
			`*\powershell.exe`,
			`*\cmd.exe`,
			`*\rundll32.exe`,
			`*\cscript.exe`,
			`*\wscript.exe`,
			`*\wmiprvse.exe`,
		},
	},
	"selection3": map[string]interface{}{
		"CommandLine": `+R +H +S +A *.cui`,
	},
}

var detection1_positive = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `+R +H +A asd.cui`,
		"ParentImage": `C:\test\wmiprvse.exe`,
	},
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `aaa`,
		"ParentImage": `C:\test\wmiprvse.exe`,
	},
}

var detection1_negative = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `+R +H +S +A lll.cui`,
		"ParentImage": `C:\test\mshta.exe`,
	},
}

var detection2 = map[string]interface{}{
	"condition": "selection1 or selection2",
	"selection1": map[string]interface{}{
		"Image": []string{
			`*\schtasks.exe`,
			`*\nslookup.exe`,
			`*\certutil.exe`,
			`*\bitsadmin.exe`,
			`*\mshta.exe`,
		},
	},
	"selection2": map[string]interface{}{
		"ParentImage": []string{
			`*\mshta.exe`,
			`*\powershell.exe`,
			`*\cmd.exe`,
			`*\rundll32.exe`,
			`*\cscript.exe`,
			`*\wscript.exe`,
			`*\wmiprvse.exe`,
		},
	},
}

var detection2_positive = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"ParentImage": `C:\test\wmiprvse.exe`,
	},
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"ParentImage": `C:\test\aaa.exe`,
	},
	map[string]string{
		"Image":       `C:\test\bbb.exe`,
		"ParentImage": `C:\test\wmiprvse.exe`,
	},
}

var detection2_negative = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bbb.exe`,
		"ParentImage": `C:\trololo\zzz.ini`,
	},
}

var detection3 = map[string]interface{}{
	"condition": "selection1 or not selection2",
	"selection1": map[string]interface{}{
		"Image": []string{
			`*\schtasks.exe`,
			`*\nslookup.exe`,
			`*\certutil.exe`,
			`*\bitsadmin.exe`,
			`*\mshta.exe`,
		},
	},
	"selection2": map[string]interface{}{
		"ParentImage": []string{
			`*\mshta.exe`,
			`*\powershell.exe`,
			`*\cmd.exe`,
			`*\rundll32.exe`,
			`*\cscript.exe`,
			`*\wscript.exe`,
			`*\wmiprvse.exe`,
		},
	},
}

var detection3_positive = []map[string]string{
	map[string]string{
		"Image":       `D:\test\bitsadmin.exe`,
		"ParentImage": `C:\test\aaa.exe`,
	},
	map[string]string{
		"Image":       `D:\test\zzz.exe`,
		"ParentImage": `C:\test\ttt.ini`,
	},
}

var detection3_negative = []map[string]string{
	map[string]string{
		"Image":       `D:\test\aaa.exe`,
		"ParentImage": `C:\test\mshta.exe`,
	},
}

var detection4 = map[string]interface{}{
	"condition": "selection1 and not selection2 and not selection3",
	"selection1": map[string]interface{}{
		"Image": []string{
			`*\schtasks.exe`,
			`*\nslookup.exe`,
			`*\certutil.exe`,
			`*\bitsadmin.exe`,
			`*\mshta.exe`,
		},
	},
	"selection2": map[string]interface{}{
		"ParentImage": []string{
			`*\mshta.exe`,
			`*\powershell.exe`,
			`*\cmd.exe`,
			`*\rundll32.exe`,
			`*\cscript.exe`,
			`*\wscript.exe`,
			`*\wmiprvse.exe`,
		},
	},
	"selection3": map[string]interface{}{
		"CommandLine": `+R +H +S +A *.cui`,
	},
}

var detection4_positive = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"ParentImage": `C:\totallylegit\firefox.exe`,
		"CommandLine": `+R +H +A asd.txt`,
	},
	map[string]string{
		"Image":       `C:\test\nslookup.exe`,
		"ParentImage": `C:\dropper\python.exe`,
		"CommandLine": `--help`,
	},
}

var detection4_negative = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `+R +H +S +A lll.cui`,
		"ParentImage": `C:\test\mshta.exe`,
	},
}

var detection5 = map[string]interface{}{
	"condition": "selection1 or not (selection3 or not (selection4 and selection5)) or selection2",
	"selection1": map[string]interface{}{
		"Field1": `aaa`,
	},
	"selection2": map[string]interface{}{
		"Field2": `bbb`,
	},
	"selection3": map[string]interface{}{
		"Field3": `ccc`,
	},
	"selection4": map[string]interface{}{
		"Field4": `ddd`,
	},
	"selection5": map[string]interface{}{
		"Field5": `eee`,
	},
}

type testCase struct {
	Rule               map[string]interface{}
	Positive, Negative []map[string]string
}

var testCases = []testCase{
	testCase{
		Rule:     detection1,
		Positive: detection1_positive,
		Negative: detection1_negative,
	},
	testCase{
		Rule:     detection2,
		Positive: detection2_positive,
		Negative: detection2_negative,
	},
	testCase{
		Rule:     detection3,
		Positive: detection3_positive,
		Negative: detection3_negative,
	},
	testCase{
		Rule:     detection4,
		Positive: detection4_positive,
		Negative: detection4_negative,
	},
	testCase{Rule: detection5},
}

func TestParse(t *testing.T) {

	for j, c := range testCases {
		parser, err := Parse(c.Rule)
		if err != nil {
			t.Fatal(err)
		}
		if c.Positive != nil {
			for i, positive := range c.Positive {
				if !parser.Match(dummyObject(positive)) {
					t.Fatalf("%d positive case %d failed to match", j, i)
				}
			}
		}
		if c.Negative != nil {
			for i, negative := range c.Negative {
				if parser.Match(dummyObject(negative)) {
					t.Fatalf("%d negative case %d matched but should not have", j, i)
				}
			}
		}
	}
}

var invalidConditions = []string{
	"selection keyword",
	"all of 1 of",
	"or and)",
}
