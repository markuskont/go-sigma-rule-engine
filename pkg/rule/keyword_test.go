package rule

import (
	"encoding/json"
	"testing"
)

var kw_example1 = `[
		"Connection refused: too many sessions for this address.",
		"Connection refused: tcp_wrappers denial.",
		"Bad HTTP verb.",
		"port and pasv both active",
		"pasv and port both active",
		"Transfer done (but failed to open directory).",
		"Could not set file modification time.",
		"bug: pid active in ptrace_sandbox_free",
		"PTRACE_SETOPTIONS failure",
		"weird status:",
		"couldn't handle sandbox event",
		"syscall * out of bounds",
		"syscall not permitted:",
		"syscall validate failed:",
		"Input line too long.",
		"poor buffer accounting in str_netfd_alloc",
		"vsf_sysutil_read_loop"
]`
var kw_example1_positive_case_0 = "syscall AWESOME out of bounds"

type dummyKw string

// GetMessage implements MessageGetter
func (d dummyKw) GetMessage() []string {
	return []string{string(d)}
}

// GetField returns a success status and arbitrary field content if requested map key is present
func (d dummyKw) GetField(_ string) (interface{}, bool) { return nil, false }

func TestKeyword(t *testing.T) {
	var obj []string
	if err := json.Unmarshal([]byte(kw_example1), &obj); err != nil {
		t.Fatalf("%s", err)
	}
	rule, err := NewKeyword(false, obj...)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	if !rule.Match(dummyKw(kw_example1_positive_case_0)) {
		t.Fatalf("%+v\n", rule)
	}
}

func BenchmarkKeyword(b *testing.B) {
	var obj []string
	json.Unmarshal([]byte(kw_example1), &obj)
	rule, _ := NewKeyword(false, obj...)
	for i := 0; i < b.N; i++ {
		rule.Match(dummyKw(kw_example1_positive_case_0))
	}
}
