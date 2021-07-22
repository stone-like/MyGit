package util

import (
	"errors"
	"mygit/src/database/content"
	"regexp"
	"sort"
)

func Contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func ContainsInt(s []int, e int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func Delete(s []string, i int) []string {
	s = append(s[:i], s[i+1:]...)
	n := make([]string, len(s))
	copy(n, s)
	return n
}

var ErrorMapNonExists = errors.New("map non exists")

var ErrorSliceNonExists = errors.New("slice non exists")

func DeleteFromMap(m map[string]content.Object, key string) map[string]content.Object {
	if _, ok := m[key]; !ok {
		return m
	}

	delete(m, key)
	return m
}

//reflectとか使って二つまとめたい
func DeleteFromParnet(m map[string][]string, key string) map[string][]string {
	if _, ok := m[key]; !ok {
		return m
	}

	delete(m, key)
	return m
}

func DeleteSpecificKey(s []string, key string) []string {
	for i, k := range s {
		if k == key {
			return Delete(s, i)
		}
	}
	return s
}

func SortStringSlice(s []string) {
	sort.Strings(s)

	sort.Slice(s, func(j, k int) bool {
		return len(s[j]) < len(s[k])
	})
}

func SortedMapKey(m map[string]int) []string {
	temp := make([]string, 0, len(m))
	for k, _ := range m {
		temp = append(temp, k)
	}

	sort.Strings(temp)

	return temp
}

func CheckRegExp(reg, branchName string) bool {
	return regexp.MustCompile(reg).MatchString(branchName)
}

func CheckRegExpSubString(reg, branchName string) [][]string {
	return regexp.MustCompile(reg).FindAllStringSubmatch(branchName, -1)
}
