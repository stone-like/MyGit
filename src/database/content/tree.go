package content

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"mygit/src/database/crypt"
	"strconv"
	"strings"
)

type Tree struct {
	Entries map[string]Object
	ObjId   string
	Content string
}

func (t *Tree) Type() string {
	return "tree"
}

func (t *Tree) ToString() string {
	// sort.Slice(t.Entries, func(i, j int) bool { return t.Entries[i].Name < t.Entries[j].Name })
	str := ""

	for k, v := range t.Entries {

		str += fmt.Sprintf("%s %s\x00", v.getMode(), k)
		ret, _ := crypt.CreateH40(v.GetObjId())

		str += ret
	}

	return str

}

func (t *Tree) Traverse(fn func(t *Tree)) {
	for _, v := range t.Entries {
		t, ok := v.(*Tree)

		if ok {
			t.Traverse(fn)
		}
	}

	fn(t)
}

// func (t *Tree) GenerateObjId() {
// 	for _, v := range t.Entries {
// 		t, ok := v.(*Tree)

// 		if ok {
// 			t.GenerateObjId()
// 		}
// 	}

// 	bytes := []byte(t.ToString())
// 	content := fmt.Sprintf("%s %d\x00%s", t.Type(), len(bytes), bytes)
// 	t.Content = content
// 	t.SetObjId(HashedBySha1(content))
// }

func (t *Tree) GetObjId() string {
	return t.ObjId
}

var (
	DIRECTORY_MODE = "40000"
)

func (t *Tree) getMode() string {
	return DIRECTORY_MODE
}

func (t *Tree) SetObjId(id string) {
	t.ObjId = id
}

func (t *Tree) Build(entries []*Entry) {

	for _, e := range entries {
		t.AddObject(e.ParentDirs(e.Path), e)
	}
}

func (t *Tree) AddObject(parents []string, obj Object) {
	if parents == nil {
		t.Entries[obj.Basename()] = obj
	} else {

		o, exists := t.Entries[parents[0]]

		var newParents []string
		if 1 < len(parents) {
			newParents = parents[1:]
		}

		if exists {
			t, ok := o.(*Tree)
			if ok {
				t.AddObject(newParents, obj)
			}
		} else {
			m := make(map[string]Object)
			newT := &Tree{Entries: m}
			newT.AddObject(newParents, obj)
			t.Entries[parents[0]] = newT
		}

	}
}

func (t *Tree) Basename() string {
	return ""
}

func GenerateTree() *Tree {
	return &Tree{
		Entries: make(map[string]Object),
	}
}

var ENTRY_SIZE = 20

func (t *Tree) Parse(r io.Reader) error {
	b := bufio.NewReader(r)

	for {
		//eofまで下三つを繰り返す
		mode, err := b.ReadString(' ')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		intMode, err := strconv.ParseInt(strings.TrimSpace(mode), 8, 64)
		if err != nil {
			return err
		}

		pathWithNullTerm, err := b.ReadString('\x00')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		path := pathWithNullTerm[:len(pathWithNullTerm)-1]

		//20bytes読むとき
		sum := make([]byte, ENTRY_SIZE)
		err = binary.Read(b, binary.BigEndian, &sum)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		hexObjIdString := hex.EncodeToString(sum)

		e := &Entry{
			ObjId: hexObjIdString,
			Mode:  int(intMode),
			Path:  path,
		}
		t.Entries[path] = e
	}

	return nil

}
