package sigma

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/markuskont/datamodels"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var placeholderTestYAML = `
---
%administrators%:
  - Admin1
  - superadmin91
  - rocky
%regular_users%:
  - maali
`

var placeholderRules = []string{
	`
---
detection:
  condition: "(selection1 or selection2) and selection3"
  selection1:
    EventID: 666,
  selection2
    EventID: 42
  selection3:
    UserName: %administrators%
`,
}

var placeholderTestCases = []struct {
	Name string
	Pos  []string
	Neg  []string
}{
	{
		"admin_user",
		[]string{
			`{"EventID": 666, "UserName": "Admin1"}`,
			`{"EventID": 666, "UserName": "superadmin91"}`,
		},
		[]string{
			`{"EventID": 3, "UserName": "Admin1"}`,
			`{"EventID": 96, "UserName": "superadmin91"}`,
			`{"EventID": 666, "UserName": "maali"}`,
		},
	},
}

func TestPlaceholders(t *testing.T) {
	ph := newPlaceholderHandle("NA")
	assert.Nil(t, yaml.Unmarshal([]byte(placeholderTestYAML), &ph.data))
	assert.NotEqual(t, 0, len(ph.data))

	rs := &Ruleset{
		mu:           &sync.RWMutex{},
		placeholders: ph,
		Rules:        make([]*Tree, 0),
	}
	for _, raw := range placeholderRules {
		var rule Rule
		assert.Nil(t, yaml.Unmarshal([]byte(raw), &rule))
		tree, err := NewTree(RuleHandle{Rule: rule})
		assert.Nil(t, err)
		rs.Rules = append(rs.Rules, tree)
	}
	updateRulesetPlaceholders(rs)

	for _, tt := range placeholderTestCases {
		t.Run(tt.Name, func(t *testing.T) {
			for _, event := range tt.Pos {
				var obj datamodels.Map
				assert.Nil(t, json.Unmarshal([]byte(event), &obj))
				_, ok := rs.EvalAll(obj)
				assert.True(t, ok)
			}
			for _, event := range tt.Neg {
				var obj datamodels.Map
				assert.Nil(t, json.Unmarshal([]byte(event), &obj))
				_, ok := rs.EvalAll(obj)
				assert.False(t, ok)
			}
		})
	}
}
