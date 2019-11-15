package types

import (
	"fmt"
	"strings"
)

type RawRule struct {
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

func (r RawRule) Condition() (string, error) {
	if r.Detection == nil || len(r.Detection) == 0 {
		return "", fmt.Errorf("missing detection key")
	}
	if val, ok := r.Detection["condition"].(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("condition key missing or not a string value")
}

func (r RawRule) GetCondition() string {
	if c, err := r.Condition(); err == nil {
		return c
	}
	return ""
}

type SearchExprType int

const (
	ExprUnk SearchExprType = iota
	ExprSelection
	ExprKeywords
)

type SearchExpr struct {
	Name    string
	Type    SearchExprType
	Content interface{}
}

func (s *SearchExpr) Guess() *SearchExpr {
	if strings.HasPrefix(s.Name, "keyword") {
		s.Type = ExprKeywords
	} else {
		s.Type = ExprSelection
	}
	return s
}

type Detection map[string]interface{}

func (d Detection) Fields() <-chan SearchExpr {
	tx := make(chan SearchExpr, 0)
	go func() {
		defer close(tx)
		for k, v := range d {
			if k != "condition" {
				e := SearchExpr{
					Name:    k,
					Content: v,
				}
				tx <- *e.Guess()
			}
		}
	}()
	return tx
}

func (d Detection) FieldSlice() []string {
	tx := make([]string, 0)
	rx := d.Fields()
	for item := range rx {
		tx = append(tx, item.Name)
	}
	return tx
}

func (d Detection) Get(key string) *SearchExpr {
	if val, ok := d[key]; ok {
		return &SearchExpr{
			Name:    key,
			Content: val,
		}
	}
	return nil
}
