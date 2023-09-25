package sigma

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// RuleHandle is a meta object containing all fields from raw yaml, but is enhanced to also
// hold debugging info from the tool, such as source file path, etc
type RuleHandle struct {
	Rule

	Path         string `json:"path"`
	Multipart    bool   `json:"multipart"`
	NoCollapseWS bool   `json:"noCollapseWS"`
}

// Rule defines raw rule conforming to sigma rule specification
// https://github.com/Neo23x0/sigma/wiki/Specification
// only meant to be used for parsing yaml that matches Sigma rule definition
type Rule struct {
	Author         string   `yaml:"author" json:"author"`
	Description    string   `yaml:"description" json:"description"`
	Falsepositives []string `yaml:"falsepositives" json:"falsepositives"`
	Fields         []string `yaml:"fields" json:"fields"`
	ID             string   `yaml:"id" json:"id"`
	Level          string   `yaml:"level" json:"level"`
	Title          string   `yaml:"title" json:"title"`
	Status         string   `yaml:"status" json:"status"`
	References     []string `yaml:"references" json:"references"`

	Logsource `yaml:"logsource" json:"logsource"`
	Detection `yaml:"detection" json:"detection"`
	Tags      `yaml:"tags" json:"tags"`
}

// HasTags returns true if the rule contains all provided tags, otherwise false
func (r *Rule) HasTags(tags []string) bool {
	lookup := make(map[string]bool, len(r.Tags))
	for _, tag := range r.Tags {
		lookup[tag] = true
	}
	for _, tag := range tags {
		if _, ok := lookup[tag]; !ok {
			return false
		}
	}
	return true
}

// RuleFromYAML parses yaml data into Rule object
func RuleFromYAML(data []byte) (r Rule, err error) {
	err = yaml.Unmarshal(data, &r)
	return
}

// IsMultipart checks if rule is multipart
func IsMultipart(data []byte) bool {
	return !bytes.HasPrefix(data, []byte("---")) && bytes.Contains(data, []byte("---"))
}

// NewRuleList 	reads a list of sigma rule paths and parses them to rule objects
func NewRuleList(files []string, skip, noCollapseWS bool, tags []string) ([]RuleHandle, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("missing rule file list")
	}
	errs := make([]ErrParseYaml, 0)
	rules := make([]RuleHandle, 0)
loop:
	for i, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		r, err := RuleFromYAML(data)
		if err != nil {
			if skip {
				errs = append(errs, ErrParseYaml{
					Path:  path,
					Count: i,
					Err:   err,
				})
				continue loop
			}
			return nil, &ErrParseYaml{Err: err, Path: path}
		}

		if !r.HasTags(tags) {
			continue loop
		}

		rules = append(rules, RuleHandle{
			Path:         path,
			Rule:         r,
			NoCollapseWS: noCollapseWS,
			Multipart:    IsMultipart(data),
		})
	}
	return rules, func() error {
		if len(errs) > 0 {
			return ErrBulkParseYaml{Errs: errs}
		}
		return nil
	}()
}

// Logsource represents the logsource field in sigma rule
// It defines relevant event streams and is used for pre-filtering
type Logsource struct {
	Product    string `yaml:"product" json:"product"`
	Category   string `yaml:"category" json:"category"`
	Service    string `yaml:"service" json:"service"`
	Definition string `yaml:"definition" json:"definition"`
}

// Detection represents the detection field in sigma rule
// contains condition expression and identifier fields for building AST
type Detection map[string]interface{}

func (d Detection) Extract() map[string]interface{} {
	tx := make(map[string]interface{})
	for k, v := range d {
		if k != "condition" {
			tx[k] = v
		}
	}
	return tx
}

// Tags contains a metadata list for tying positive matches together with other threat intel sources
// For example, for attaching MITRE ATT&CK tactics or techniques to the event
type Tags []string

// Result is an object returned on positive sigma match
type Result struct {
	Tags `json:"tags"`

	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Author string `json:"author"`
}

// Results should be returned when single event matches multiple rules
type Results []Result

// NewRuleFileList finds all yaml files from defined root directories
// Subtree is scanned recursively
// No file validation, other than suffix matching
func NewRuleFileList(dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, errors.New("rule directories undefined")
	}
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
			return out, err
		}
	}
	return out, nil
}
