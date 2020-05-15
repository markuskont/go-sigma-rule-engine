package sigma

import "fmt"

// Config is used as argument to creating a new ruleset
type Config struct {
	// root directory for recursive rule search
	// rules must be readable files with "yml" suffix
	Directory []string
	// by default, a rule parse fail will simply increment Ruleset.Failed counter when failing to
	// parse yaml or rule AST
	// this parameter will cause an early error return instead
	FailOnRuleParse, FailOnYamlParse bool
}

func (c Config) validate() error {
	if c.Directory == nil || len(c.Directory) == 0 {
		return fmt.Errorf("Missing root directory for sigma rules")
	}
	return nil
}

// Ruleset is a collection of rules
type Ruleset struct {
	Rules []*Tree
	root  []string

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
