package util

import (
	"reflect"
	"sort"
)

func SortedKeys(mapInt interface{}) []string {
	values := reflect.ValueOf(mapInt).MapKeys()
	result := make([]string, len(values))
	for i, value1 := range values {
		result[i] = value1.String()
	}
	sort.Strings(result)
	return result
}

func SortKeysReverse(keys []string) []string {
	for i := 0; i < len(keys)/2; i++ {
		keys[i], keys[len(keys)-i-1] = keys[len(keys)-i-1], keys[i]
	}

	return keys
}
