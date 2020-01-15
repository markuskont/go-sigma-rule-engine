package sigma

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ccdcoe/go-peek/pkg/utils"
	"gopkg.in/yaml.v2"
)

/*
	This package will be the main entrypoint when imported from other projects.
*/

type Config struct {
	Directories []string
}

func (c *Config) Validate() error {
	var err error
	if c.Directories == nil || len(c.Directories) == 0 {
		return fmt.Errorf("Missing sigma rule directory")
	}
	for i, dir := range c.Directories {
		if dir, err = utils.ExpandHome(dir); err != nil {
			return err
		} else {
			c.Directories[i] = dir
		}
	}
	return nil
}

type UnsupportedRawRule struct {
	Path   string
	Reason string
	Error  error
	data   []byte
	Rule   *RawRule
}

func (u UnsupportedRawRule) Raw() string {
	if u.data == nil || len(u.data) == 0 {
		return fmt.Sprintf("missing data for unsupported rule %s", u.Path)
	}
	return string(u.data)
}

type Rule struct {
	tree *Tree
	RawRule
	Path string
}

// SafeRuleMap is thread safe storage for RuleGroups
type SafeRuleMap sync.Map

type RuleGroup []Rule

type Ruleset struct {
	dirs []string

	Rules map[string]RuleGroup

	Total       int
	Unsupported []UnsupportedRawRule
	Broken      []UnsupportedRawRule
}

func NewRuleset(c *Config) (*Ruleset, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r := &Ruleset{
		dirs:        c.Directories,
		Rules:       make(map[string]RuleGroup),
		Unsupported: make([]UnsupportedRawRule, 0),
		Broken:      make([]UnsupportedRawRule, 0),
	}
	files, err := discoverRuleFilesInDir(r.dirs)
	if err != nil {
		return nil, err
	}
	decoded := make([]Rule, 0)
loop:
	for _, path := range files {
		data, err := ioutil.ReadFile(path) // just pass the file name
		if err != nil {
			return nil, err
		}
		if bytes.Contains(data, []byte("---")) {
			r.Unsupported = append(r.Unsupported, UnsupportedRawRule{
				Path:   path,
				Reason: "Multi-part YAML",
				Error:  nil,
			})
			continue loop
		}
		var s RawRule
		if err := yaml.Unmarshal([]byte(data), &s); err != nil {
			return nil, err
		}
		decoded = append(decoded, Rule{
			RawRule: s,
			Path:    path,
		})
	}
	rules := make([]Rule, 0)

decodedloop:
	for _, dec := range decoded {
		tree, err := ParseDetection(dec.Detection)
		if err != nil {
			switch err.(type) {
			case *ErrUnsupportedToken, *ErrIncompleteDetection, *ErrWip, ErrUnsupportedToken, ErrIncompleteDetection, ErrWip:
				r.Unsupported = append(r.Unsupported, UnsupportedRawRule{
					Path:  dec.Path,
					Rule:  &dec.RawRule,
					Error: err,
				})
				continue decodedloop
			default:
				r.Broken = append(r.Broken, UnsupportedRawRule{
					Path:  dec.Path,
					Rule:  &dec.RawRule,
					Error: err,
				})
				continue decodedloop
			}
		}
		rules = append(rules, Rule{
			tree:    tree,
			RawRule: dec.RawRule,
			Path:    dec.Path,
		})
	}
	if len(rules) == 0 {
		return r, fmt.Errorf("unable to parse any rules from %+v", r.dirs)
	}
	r.Total = len(rules)

groupLoop:
	for _, rule := range rules {
		if rule.Logsource.Product == "" {
			r.Unsupported = append(r.Unsupported, UnsupportedRawRule{
				Rule:   &rule.RawRule,
				Reason: "Missing PRODUCT in LOGSOURCE",
				Path:   rule.Path,
			})
			r.Total--
			continue groupLoop
		}
		if val, ok := r.Rules[rule.Logsource.Product]; ok {
			r.Rules[rule.Logsource.Product] = append(val, rule)
		} else {
			r.Rules[rule.Logsource.Product] = make(RuleGroup, 1)
			r.Rules[rule.Logsource.Product][0] = rule
		}
	}
	return r, nil
}

func discoverRuleFilesInDir(dirs []string) ([]string, error) {
	out := make([]string, 0)
	for _, dir := range dirs {
		if err := filepath.Walk(dir, func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if !info.IsDir() && strings.HasSuffix(path, "yml") {
				out = append(out, path)
			}
			return err
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}
