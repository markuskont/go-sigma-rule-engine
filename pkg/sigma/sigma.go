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

type UnsupportedRawRule struct {
	Path   string
	Reason string
	Error  error
	data   []byte
}

func (u UnsupportedRawRule) Raw() string {
	if u.data == nil || len(u.data) == 0 {
		return fmt.Sprintf("missing data for unsupported rule %s", u.Path)
	}
	return string(u.data)
}

type Rule struct {
	Tree
	RawRule
}

type Ruleset struct {
	dirs []string

	Unsupported []UnsupportedRawRule
}

func NewRuleset(c *Config) (*Ruleset, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r := &Ruleset{
		dirs:        c.Direcotries,
		Unsupported: make([]UnsupportedRawRule, 0),
	}
	files, err := discoverRuleFilesInDir(r.dirs)
	if err != nil {
		return nil, err
	}
	decoded := make([]RawRule, 0)
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
		decoded = append(decoded, s)
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
