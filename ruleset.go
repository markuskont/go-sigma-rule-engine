package sigma

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
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
	// by default, we will collapse whitespace for both rules and data of non-regex rules and non-regex compared data
	//setthig this to true turns that behavior off
	NoCollapseWS bool
	// filesystem path to YAML file containing placeholders
	PlaceholderYAML string
}

func (c Config) validate() error {
	if c.Directory == nil || len(c.Directory) == 0 {
		return fmt.Errorf("missing root directory for sigma rules")
	}
	for _, dir := range c.Directory {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", dir)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
	}
	return nil
}

// Ruleset is a collection of rules
type Ruleset struct {
	mu *sync.RWMutex

	Rules []*Tree
	root  []string

	placeholders *placeholderHandle

	Total, Ok, Failed, Unsupported int
}

// NewRuleset instanciates a Ruleset object
func NewRuleset(c Config) (*Ruleset, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	files, err := NewRuleFileList(c.Directory)
	if err != nil {
		return nil, err
	}
	var fail, unsupp int
	rules, err := NewRuleList(files, !c.FailOnYamlParse, c.NoCollapseWS)
	if err != nil {
		switch e := err.(type) {
		case ErrBulkParseYaml:
			fail += len(e.Errs)
		default:
			return nil, err
		}
	}
	rs := &Ruleset{
		mu:          &sync.RWMutex{},
		root:        c.Directory,
		Failed:      fail,
		Unsupported: unsupp,
		Total:       len(files),
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

	rs.Rules = set
	rs.Ok = len(set)

	if c.PlaceholderYAML != "" {
		rs.placeholders = newPlaceholderHandle(c.PlaceholderYAML)
		if err := rs.ReloadPlaceholders(); err != nil {
			return rs, err
		}
	}
	return rs, nil
}

func (r *Ruleset) EvalAll(e Event) (Results, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	results := make(Results, 0)
	for _, rule := range r.Rules {
		if res, match := rule.Eval(e); match {
			results = append(results, *res)
		}
	}
	if len(results) > 0 {
		return results, true
	}
	return nil, false
}

func (r *Ruleset) ReloadPlaceholders() error {
	if err := r.placeholders.load(); err != nil {
		return err
	}
	updateRulesetPlaceholders(r)
	return nil
}

func (r *Ruleset) RunPlaceholderReload(ctx context.Context, d time.Duration, errFn func(error)) error {
	if r.placeholders == nil {
		return errors.New("cannot initialize placeholder reload, not initialized")
	}
	return r.placeholders.runLoader(ctx, d, errFn)
}
