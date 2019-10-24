package rule

import (
	"encoding/json"
	"testing"
)

var example1 = `{
	"UserAgent": [
		"python*",
		"*urllib*",
		"webclient",
		"Mozilla/4.0",
		"Netscape",
		"hots scot"
	]
} `
var example1_valid = `{
	"UserAgent": "Python-urllib/2.5"
}`
var example2 = `{
	"Image": [
		"*\\schtasks.exe",
		"*\\nslookup.exe",
		"*\\certutil.exe",
		"*\\bitsadmin.exe",
		"*\\mshta.exe"
	],
	"ParentImage": [
		"*\\mshta.exe",
		"*\\powershell.exe",
		"*\\cmd.exe",
		"*\\rundll32.exe",
		"*\\cscript.exe",
		"*\\wscript.exe",
		"*\\wmiprvse.exe"
	]
}`
var example2_valid = `{
	"Image": "BLABLA\\taylor.switf\\nslookup.exe",
	"ParentImage": "WIN32\\powershell.exe"
}`

var example3 = `{
	"CommandLine": "+R +H +S +A \\\\*.cui",
	"ParentCommandLine": "C:\\WINDOWS\\system32\\\\*.bat",
	"ParentImage": "*\\cmd.exe"
}`
var example3_valid = `{
	"CommandLine": "+R +H +S +A \\\\somecrapgoeshere.cui",
	"ParentCommandLine": "C:\\WINDOWS\\system32\\\\myawesomemalwarescript.bat",
	"ParentImage": "%WIN32\\cmd.exe"
}`
var example4 = `{
	"EventID": 5136,
	"LDAPDisplayName": "ntSecurityDescriptor",
	"Value": [
		"*1131f6ad-9c07-11d1-f79f-00c04fc2dcd2*",
		"*1131f6aa-9c07-11d1-f79f-00c04fc2dcd2*"
	]
}`
var example4_valid = `{
	"EventID": 5136,
	"LDAPDisplayName": "ntSecurityDescriptor",
	"Value": "BBBBB1131f6ad-9c07-11d1-f79f-00c04fc2dcd2AAAAA"
}`
var example5 = `{
	"DestinationIp": [
		"10.*",
		"192.168.*",
		"172.16.*",
		"172.17.*",
		"172.18.*",
		"172.19.*",
		"172.20.*",
		"172.21.*",
		"172.22.*",
		"172.23.*",
		"172.24.*",
		"172.25.*",
		"172.26.*",
		"172.27.*",
		"172.28.*",
		"172.29.*",
		"172.30.*",
		"172.31.*",
		"127.0.0.1"
	],
	"DestinationIsIpv6": "false",
	"User": "NT AUTHORITY\\SYSTEM"
}`
var example5_valid = `{
	"DestinationIp": "10.0.0.14",
	"DestinationIsIpv6": "false",
	"User": "NT AUTHORITY\\SYSTEM"
}`
var example6 = `{
	"SomeIDField": [
		666,
		13
	],
	"OtherID": 42,
	"StringNumber": 42,
	"ActualText": "message * aaa"
}`
var example6_valid = `{
	"SomeIDField": 666,
	"OtherID": 42,
	"StringNumber": "42",
	"ActualText": "message from aaa"
}`

type inputExample struct {
	raw_rule            string
	raw_map             map[string]interface{}
	raw_true_positive   []string
	event_true_positive []dummyObject
	rule                *Fields
}

type dummyObject map[string]interface{}

// GetMessage implements MessageGetter
func (d dummyObject) GetMessage() []string {
	return []string{}
}

// GetField returns a success status and arbitrary field content if requested map key is present
func (d dummyObject) GetField(key string) (interface{}, bool) {
	if val, ok := d[key]; ok {
		return val, ok
	}
	return nil, false
}

var (
	inputs = []*inputExample{
		&inputExample{raw_rule: example1, raw_true_positive: []string{example1_valid}},
		&inputExample{raw_rule: example2, raw_true_positive: []string{example2_valid}},
		&inputExample{raw_rule: example3, raw_true_positive: []string{example3_valid}},
		&inputExample{raw_rule: example4, raw_true_positive: []string{example4_valid}},
		&inputExample{raw_rule: example5, raw_true_positive: []string{example5_valid}},
		&inputExample{raw_rule: example6, raw_true_positive: []string{example6_valid}},
	}
)

func TestInputParse(t *testing.T) {
	for _, in := range inputs {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(in.raw_rule), &obj); err != nil {
			t.Fatalf("%s, %s", in.raw_rule, err)
		}
		in.raw_map = obj
		in.event_true_positive = make([]dummyObject, len(in.raw_true_positive))
		for i, item := range in.raw_true_positive {
			var testCase dummyObject
			if err := json.Unmarshal([]byte(item), &testCase); err != nil {
				t.Fatalf("%s, %s", item, err)
			}
			in.event_true_positive[i] = testCase
		}
	}
	for _, in := range inputs {
		if in.raw_map == nil || len(in.raw_map) == 0 {
			t.Fail()
		}
		rule, err := NewFields(in.raw_map, false, true)
		if err != nil {
			t.Fatalf("%s | %s", in.raw_rule, err)
		}
		in.rule = rule
	}
	for _, in := range inputs {
		for _, testCase := range in.event_true_positive {
			if !in.rule.Match(testCase) {
				t.Fatalf("condition %s failed for %s", in.raw_rule, in.raw_true_positive)
			}
		}
	}
}

func BenchmarkParse(b *testing.B) {
	for _, in := range inputs {
		var obj map[string]interface{}
		json.Unmarshal([]byte(in.raw_rule), &obj)
		in.raw_map = obj
		in.event_true_positive = make([]dummyObject, len(in.raw_true_positive))
		for i, item := range in.raw_true_positive {
			var testCase dummyObject
			json.Unmarshal([]byte(item), &testCase)
			in.event_true_positive[i] = testCase
		}
		rule, _ := NewFields(in.raw_map, false, true)
		in.rule = rule
		for i, item := range in.raw_true_positive {
			var testCase dummyObject
			json.Unmarshal([]byte(item), &testCase)
			in.event_true_positive[i] = testCase
		}
	}
	c := 3
	for i := 0; i < b.N; i++ {
		inputs[c].rule.Match(inputs[c].event_true_positive[0])
	}
}
