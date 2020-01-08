package condition

import (
	"fmt"
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

var detection1 = map[string]interface{}{
	"condition": "not selection1",
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
	"selection2": map[string]interface{}{
		//"CommandLine": `+R +H +S +A *.cui`,
		"CommandLine": `aaa`,
	},
}

var detection1_positive = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `+R +H +A asd.cui`,
		"ParentImage": `C:\test\wmiprvse.exe`,
	},
}

var detection1_negative = []map[string]string{
	map[string]string{
		"Image":       `C:\test\bitsadmin.exe`,
		"CommandLine": `+R +H +S +A asd.cui`,
		"ParentImage": `C:\test\bbb.exe`,
	},
	/*
		map[string]string{
			"Image":       `C:\test\aaa.exe`,
			"CommandLine": `+R +H +A asd.cui`,
			"ParentImage": `C:\test\lll.exe`,
		},
	*/
}

func TestParse(t *testing.T) {
	parser, err := Parse(detection1)
	if err != nil {
		t.Fatal(err)
	}
	for i, positive := range detection1_positive {
		if !parser.Match(dummyObject(positive)) {
			t.Fatalf("positive case %d failed to match", i)
		}
	}
	/*
		for i, negative := range detection1_negative {
			if parser.Match(dummyObject(negative)) {
				t.Fatalf("negative case %d matched but should not have", i)
			}
		}
	*/
}

var invalidConditions = []string{
	"selection keyword",
	"all of 1 of",
	"or and)",
}
