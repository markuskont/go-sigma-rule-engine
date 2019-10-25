package types

import "fmt"

type RawRule struct {
	// Unique identifier that will be attached to positive match
	ID int `yaml:"id" json:"id"`

	// https://github.com/Neo23x0/sigma/wiki/Specification
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

	Detection map[string]interface{} `yaml:"detection" json:"detection"`

	Fields         interface{} `yaml:"fields" json:"fields"`
	Falsepositives interface{} `yaml:"falsepositives" json:"falsepositives"`
	Level          interface{} `yaml:"level" json:"level"`
	Tags           []string    `yaml:"tags" json:"tags"`
}

func (r RawRule) Condition() (string, error) {
	if r.Detection == nil || len(r.Detection) == 0 {
		return "", fmt.Errorf("missing detection key")
	}
	if val, ok := r.Detection["condition"].(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("condition key missing or not a string value")
}
