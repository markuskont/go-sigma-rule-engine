package sigma

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccdcoe/go-peek/pkg/utils"
	"gopkg.in/yaml.v2"
)

/*
	This package will be the main entrypoint when imported from other projects.
*/

type Config struct {
	Direcotries []string
}

func (c *Config) Validate() error {
	var err error
	if c.Direcotries == nil || len(c.Direcotries) == 0 {
		return fmt.Errorf("Missing sigma rule directory")
	}
	for i, dir := range c.Direcotries {
		if dir, err = utils.ExpandHome(dir); err != nil {
			return err
		} else {
			c.Direcotries[i] = dir
		}
	}
	return nil
}

type UnsupportedRule struct {
	Path   string
	Reason string
	data   []byte
}

func (u UnsupportedRule) Raw() string {
	if u.data == nil || len(u.data) == 0 {
		return fmt.Sprintf("missing data for unsupported rule %s", u.Path)
	}
	return string(u.data)
}

type Ruleset struct {
	dirs []string

	Unsupported []UnsupportedRule
}

func NewRuleset(c *Config) (*Ruleset, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r := &Ruleset{
		dirs:        c.Direcotries,
		Unsupported: make([]UnsupportedRule, 0),
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
			r.Unsupported = append(r.Unsupported, UnsupportedRule{
				Path:   path,
				Reason: "Multi-part YAML",
				data:   data,
			})
			continue loop
		}
		var s Rule
		if err := yaml.Unmarshal([]byte(data), &s); err != nil {
			return nil, err
		}
		decoded = append(decoded, s)
	}
	return r, nil
}

type Rule struct {
	File string `yaml:"file" json:"file"`

	// https://github.com/Neo23x0/sigma/wiki/Specification
	ID          string `yaml:"id" json:"id"`
	Title       string `yaml:"title" json:"title"`
	Status      string `yaml:"status" json:"status"`
	Description string `yaml:"description" json:"description"`
	Author      string `yaml:"author" json:"author"`
	// A list of URL-s to external sources
	References []string `yaml:"references" json:"references"`
	Logsource  struct {
		Product    string `yaml:"product" json:"product"`
		Category   string `yaml:"category" json:"category"`
		Service    string `yaml:"service" json:"service"`
		Definition string `yaml:"definition" json:"definition"`
	} `yaml:"logsource" json:"logsource"`

	Detection Detection `yaml:"detection" json:"detection"`

	Fields         interface{} `yaml:"fields" json:"fields"`
	Falsepositives interface{} `yaml:"falsepositives" json:"falsepositives"`
	Level          interface{} `yaml:"level" json:"level"`
	Tags           []string    `yaml:"tags" json:"tags"`
}

func (r Rule) Condition() (string, error) {
	if r.Detection == nil || len(r.Detection) == 0 {
		return "", fmt.Errorf("missing detection key")
	}
	if val, ok := r.Detection["condition"].(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("condition key missing or not a string value")
}

func (r Rule) GetCondition() string {
	if c, err := r.Condition(); err == nil {
		return c
	}
	return ""
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
