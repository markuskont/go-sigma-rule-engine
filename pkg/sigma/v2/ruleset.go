package sigma

import (
	"container/list"
	"fmt"
	"time"
)

// Config is used as argument to creating a new ruleset
type Config struct {
	// root directory for recursive rule search
	// rules must be readable files with "yml" suffix
	Directory []string
	// by default, a rule parse fail will simply increment Ruleset.Failed counter when failing to
	// parse yaml or rule AST
	// this parameter will cause an early error return instead
	FailOnRuleParse, FailOnYamlParse bool

	// Enable rule profiling
	// Only for ruleset debugging and benchmarking
	// Collects RuleProfileItem objects into linked lists per rule and worker
	// Needs to be either flushed periodically or limited to small test iterations
	// Does map lookups on each rule match, so will reduce overall throughput
	Profile bool
}

func (c Config) validate() error {
	if c.Directory == nil || len(c.Directory) == 0 {
		return fmt.Errorf("Missing root directory for sigma rules")
	}
	return nil
}

type RuleProfileItem struct {
	Took  time.Duration `json:"took"`
	Match bool          `json:"match"`
}

type RuleProfile struct {
	Enabled      bool
	Measurements map[string]*list.List
}

func (p RuleProfile) DrainRule(uuid string) (<-chan RuleProfileItem, bool) {
	rule, ok := p.Measurements[uuid]
	if !ok {
		return nil, false
	}
	tx := make(chan RuleProfileItem, 0)
	go func() {
		defer close(tx)
		for e := rule.Front(); e != nil; e = e.Next() {
			tx <- e.Value.(RuleProfileItem)
		}
	}()
	return tx, true
}

func newRuleProfile(enabled bool, set []*Tree) *RuleProfile {
	return &RuleProfile{
		Enabled: enabled,
		Measurements: func() map[string]*list.List {
			profile := make(map[string]*list.List)
			for _, rule := range set {
				profile[rule.Rule.ID] = list.New()
			}
			return profile
		}(),
	}
}

// Ruleset is a collection of rules
type Ruleset struct {
	Rules []*Tree
	root  []string

	RuleProfile

	Total, Ok, Failed, Unsupported int
}

// NewRuleset instanciates a Ruleset object
func NewRuleset(c Config) (*Ruleset, error) {
	files, err := NewRuleFileList(c.Directory)
	if err != nil {
		return nil, err
	}
	var fail, unsupp int
	rules, err := NewRuleList(files, !c.FailOnYamlParse)
	if err != nil {
		switch e := err.(type) {
		case ErrBulkParseYaml:
			fail += len(e.Errs)
		default:
			return nil, err
		}
	}
	set := make([]*Tree, 0)
loop:
	for _, raw := range rules {
		if raw.Multipart {
			unsupp++
			continue loop
		}
		tree, err := NewTree(raw)
		if err != nil {
			switch err.(type) {
			case ErrUnsupportedToken, *ErrUnsupportedToken:
				unsupp++
			default:
				fail++
			}
			continue loop
		}
		set = append(set, tree)
	}
	return &Ruleset{
		root:        c.Directory,
		Rules:       set,
		Failed:      fail,
		Ok:          len(set),
		Unsupported: unsupp,
		Total:       len(files),
		RuleProfile: *newRuleProfile(c.Profile, set),
	}, nil
}

func (r Ruleset) Match(e Event) bool {
	for _, rule := range r.Rules {
		if rule.Match(e) {
			return true
		}
	}
	return false
}

func (r Ruleset) EvalAll(e Event) (Results, bool) {
	results := make(Results, 0)
	for _, rule := range r.Rules {
		start := time.Now()
		res, match := rule.Eval(e)
		if match {
			results = append(results, *res)
		}
		if r.RuleProfile.Enabled {
			r.RuleProfile.Measurements[rule.Rule.ID].PushBack(
				RuleProfileItem{
					Took:  time.Since(start),
					Match: match,
				})
		}
	}
	if len(results) > 0 {
		return results, true
	}
	return nil, false
}
