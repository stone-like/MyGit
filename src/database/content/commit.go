package content

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Commit struct {
	ObjId   string
	Tree    *Tree
	Author  *Author
	Message string
	Parent  string
}

func (c *Commit) Type() string {
	return "commit"
}

func (c *Commit) ToString() string {

	var str string
	str = str + fmt.Sprintf("tree %s\n", c.Tree.GetObjId())

	if len(c.Parent) != 0 {
		str = str + fmt.Sprintf("parent %s\n", c.Parent)
	}

	str = str + fmt.Sprintf("author %s\n", c.Author.ToString())
	str = str + fmt.Sprintf("commiter %s\n", c.Author.ToString())
	str += "\n"
	str = str + c.Message + "\n"

	return str
}

func (c *Commit) GetObjId() string {
	return c.ObjId
}

func (c *Commit) SetObjId(objId string) {
	c.ObjId = objId
}

func (c *Commit) Basename() string {
	return ""
}

func (c *Commit) getMode() string {
	return ""
}

type CommitFromMem struct {
	ObjId   string
	Tree    string
	Author  *Author
	Message string
	Parent  string
}

func (c *CommitFromMem) SetObjId(objId string) {
	c.ObjId = objId
}

func (c *CommitFromMem) GetObjId() string {
	return c.ObjId
}

func (c *CommitFromMem) Type() string {
	return "commit"
}

func (c *CommitFromMem) GetFirstLineMessage() string {
	s := strings.Split(c.Message, "\n")
	return s[0]
}

func (c *CommitFromMem) Parse(r io.Reader) error {
	s := bufio.NewScanner(r)

	var messages []string
	for s.Scan() {
		text := s.Text()
		words := strings.Fields(text)

		if c.Tree != "" && c.Author != nil {
			messages = append(messages, text)
		} else if len(words) == 2 {
			//tree,parentの時

			if words[0] == "parent" {
				c.Parent = words[1]
			} else {
				//treeの時
				c.Tree = words[1]
			}
		} else if len(words) == 5 {
			//authorとcommiter
			emailLen := len(words[2])
			createdAt := words[3] + " " + words[4]

			if words[0] == "author" {
				a := &Author{
					Name:      words[1],
					Email:     words[2][1 : emailLen-1],
					CreatedAt: createdAt,
				}
				c.Author = a
			}

			s.Scan() //authorの情報だけでいいのでcommiterのparseはskip
			s.Scan() //commiterとmessageの間に改行があるのでそれもskip

		}

	}
	c.Message = strings.Join(messages, "\n")

	return nil
}
