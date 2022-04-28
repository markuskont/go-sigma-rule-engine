package sigma

import "strings"

// GetField is a helper for retreiving nested JSON keys with dot notation
func GetField(key string, data map[string]interface{}) (interface{}, bool) {
	if data == nil {
		return nil, false
	}
	bits := strings.SplitN(key, ".", 2)
	if len(bits) == 0 {
		return nil, false
	}
	if val, ok := data[bits[0]]; ok {
		switch res := val.(type) {
		case map[string]interface{}:
			return GetField(bits[1], res)
		default:
			return val, ok
		}
	}
	return nil, false
}

// DynamicMap is a reference type for implementing sigma Matcher
type DynamicMap map[string]interface{}

// Keywords implements Keyworder
func (s DynamicMap) Keywords() ([]string, bool) {
	return nil, false
}

// Select implements Selector
func (s DynamicMap) Select(key string) (interface{}, bool) {
	return GetField(key, s)
}
