package database

import (
	con "mygit/src/database/content"
	"sort"
)

type EntryKey struct {
	Path  string
	Stage int
}

type KeysSlice []EntryKey

func (k KeysSlice) Contains(key EntryKey) bool {
	for _, ek := range k {
		if ek == key {
			return true
		}
	}

	return false
}

func (k KeysSlice) Delete(key EntryKey) KeysSlice {

	var temp KeysSlice

	for _, v := range k {
		if v == key {
			continue
		} else {
			temp = append(temp, v)
		}
	}
	return temp
}

type EntriesMap map[EntryKey]con.Object

func (e EntriesMap) GetValue(path string, stage int) (con.Object, bool) {
	v, ok := e[EntryKey{Path: path, Stage: stage}]

	return v, ok
}

func (e EntriesMap) Contains(key EntryKey) bool {
	for k, _ := range e {
		if k == key {
			return true
		}
	}

	return false
}

//keyに構造体を使っていて特殊なので拡張性を考えずべた書きする
func (e EntriesMap) GetSortedkey() []EntryKey {

	var entrykeySlice = make([]EntryKey, 0, len(e))
	for k, _ := range e {
		entrykeySlice = append(entrykeySlice, k)
	}

	sort.Slice(entrykeySlice, func(i, j int) bool {

		if entrykeySlice[i].Path != entrykeySlice[j].Path {
			return entrykeySlice[i].Path < entrykeySlice[j].Path
		} else {
			return entrykeySlice[i].Stage < entrykeySlice[j].Stage
		}
	})

	return entrykeySlice

}
