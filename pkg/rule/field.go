package rule

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

// JSON numbers are by spec all float64 values
type numPatterns []float64

func (p numPatterns) match(n float64) bool {
	for _, val := range p {
		if n == val {
			return true
		}
	}
	return false
}

func (p numPatterns) matchNumber(num interface{}, tryString bool) bool {
	switch nu := num.(type) {
	case float64:
		return p.match(nu)
	case int:
		return p.match(float64(nu))
	case int64:
		return p.match(float64(nu))
	case uint:
		return p.match(float64(nu))
	case uint64:
		return p.match(float64(nu))
	case int32:
		return p.match(float64(nu))
	case uint32:
		return p.match(float64(nu))
	case string:
		if tryString {
			if val, err := strconv.Atoi(nu); err == nil {
				return p.match(float64(val))
			}
		}
		return false
	default:
		return false
	}
}

func (p numPatterns) len() int    { return len(p) }
func (p numPatterns) empty() bool { return p.len() == 0 }

type Fields struct {
	sPatterns map[string]stringPatterns
	nPatterns map[string]numPatterns

	toLower         bool
	tryStingNumbers bool
	Stats
}

func NewFields(raw map[string]interface{}, lowercase, stringnum bool) (*Fields, error) {
	if raw == nil || len(raw) == 0 {
		return nil, fmt.Errorf("wrong interface type for rule condition, only map[string]interface{} supported")
	}
	f := &Fields{
		toLower:         lowercase,
		tryStingNumbers: stringnum,
	}
	// Thank you pythonistas
	for k, v := range raw {
		switch condition := v.(type) {
		case string:
			if f.sPatterns == nil {
				f.sPatterns = make(map[string]stringPatterns)
			}
			patterns, err := newStringPatterns(condition)
			if err != nil {
				return f, err
			}
			f.sPatterns[k] = *patterns
		case float64:
			if f.nPatterns == nil {
				f.nPatterns = make(map[string]numPatterns)
			}
			f.nPatterns[k] = []float64{condition}
		case int:
			if f.nPatterns == nil {
				f.nPatterns = make(map[string]numPatterns)
			}
			f.nPatterns[k] = []float64{float64(condition)}
		case []interface{}:
			var t reflect.Kind
			var stringAndNumber bool
			// make sure all list types are the same
		loop:
			for i, item := range condition {
				if i > 0 {
					if t2 := reflect.TypeOf(item).Kind(); t2 != t && (t == reflect.String || t2 == reflect.String) {
						stringAndNumber = true
						t = reflect.String
						break loop
					} else if t2 != t {
						return f, fmt.Errorf(
							"Selection/field rule parse fail for key %s list contains %s and %s",
							k,
							t.String(),
							t2.String(),
						)
					}
				}
				t = reflect.TypeOf(item).Kind()
			}
			switch t {
			case reflect.String:
				if f.sPatterns == nil {
					f.sPatterns = make(map[string]stringPatterns)
				}
				patterns, err := newStringPatterns(func() []string {
					str := make([]string, len(condition))
					for i, item := range condition {
						// This should already be checked
						if stringAndNumber {
							switch cast := v.(type) {
							case string:
								str[i] = cast
							case int:
								str[i] = strconv.Itoa(cast)
							case float64:
								str[i] = strconv.Itoa(int(cast))
							default:

							}
						} else {
							str[i] = item.(string)
						}
					}
					return str
				}()...)
				if err != nil {
					return f, err
				}
				f.sPatterns[k] = *patterns
			case reflect.Float64:
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				f.nPatterns[k] = func() []float64 {
					flt := make([]float64, len(condition))
					for i, item := range condition {
						// this should already be checked
						flt[i] = item.(float64)
					}
					return flt
				}()
			case reflect.Int:
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				f.nPatterns[k] = func() []float64 {
					flt := make([]float64, len(condition))
					for i, item := range condition {
						// this should already be checked
						flt[i] = float64(item.(int))
					}
					return flt
				}()
			default:
				return f, fmt.Errorf("unsupported type for key %s, got %s but expected string or float64", k, t.String())
			}
		default:
			return nil, fmt.Errorf(
				"wrong rule type for [%+v], field [%s], got %T, only support string, float64, or their respective sliced versions",
				raw, k, v,
			)
		}
	}
	return f, nil
}

func (f *Fields) Match(obj types.EventChecker) bool {
	f.Total++
	if f.sPatterns != nil && len(f.sPatterns) > 0 {
	and1:
		for field, patterns := range f.sPatterns {
			if val, ok := obj.GetField(field); ok {
				if str, ok := val.(string); ok {
					if matchKeywords(patterns, f.toLower, str) {
						continue and1
					}
				}
				return false
			}
			return false
		}
	}
	if f.nPatterns != nil && len(f.nPatterns) > 0 {
	and2:
		for field, patterns := range f.nPatterns {
			if val, ok := obj.GetField(field); ok {
				if patterns.matchNumber(val, f.tryStingNumbers) {
					continue and2
				}
			}
			return false
		}
	}
	return true
}

func (f Fields) Self() interface{} { return f }
