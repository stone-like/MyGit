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

func HasKey(m interface{}, key string) bool {
	for _, k := range SortedKeys(m) {
		if k == key {
			return true
		}
	}

	return false
}

func Copy(m1, m2 interface{}) {
	m := reflect.ValueOf(m1)
	iter := reflect.ValueOf(m2).MapRange()
	for iter.Next() {
		m.SetMapIndex(iter.Key(), iter.Value())
	}
}

//map[string]struct{}としてmapをsetとして使っている
func IsContainOtherSet(targetMap, otherMap map[string]struct{}) bool {

	for k, _ := range otherMap {
		_, ok := targetMap[k]
		if !ok {
			return false
		}

	}

	return true
}
