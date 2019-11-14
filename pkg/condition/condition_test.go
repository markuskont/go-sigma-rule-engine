package condition

import (
	"fmt"
	"testing"
)

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
	"condition": "Image and not (ParentImage or CommandLine)",
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
	"CommandLine": `+R +H +S +A \\*.cui`,
}

func TestParse(t *testing.T) {
	_, err := Parse(detection1)
	if err != nil {
		t.Fatal(err)
	}
}

var invalidConditions = []string{
	"selection keyword",
	"all of 1 of",
	"or and)",
}

func TestInvalid(t *testing.T) {
	for _, str := range invalidConditions {
		fmt.Println("******************CASE: ", str)
		l := lex(str)
		for tok := range l.items {
			/*
				if tok.T == TokErr {
					panic("waat")
				}
			*/
			fmt.Println(tok)
		}
	}
}
