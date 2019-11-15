package condition

import (
	"fmt"
	"strings"
)

func interfaceMapToStringInterfaceMap(m map[interface{}]interface{}) (map[string]interface{}, error) {
	m2 := make(map[string]interface{})
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return m2, fmt.Errorf("failed to create selection rule from interface")
		}
		m2[sk] = v
	}
	return m2, nil
}

func isKeywords(s string) bool { return strings.HasPrefix(s, "keywords") }
