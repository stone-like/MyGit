package content

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Author struct {
	Name      string
	Email     string
	CreatedAt string
}

func (a *Author) ToString() string {
	return fmt.Sprintf("%s <%s> %s", a.Name, a.Email, a.CreatedAt)
}

func generateTime(t time.Time) string {
	s := t.String()

	words := strings.Fields(s)
	unixTime := t.Unix()

	return fmt.Sprintf("%d %s", unixTime, words[2])
}

func (a *Author) ShortTime() string {
	words := strings.Fields(a.CreatedAt)
	ut, _ := strconv.Atoi(words[0])
	t := time.Unix(int64(ut), 0)
	return t.Format("2006-01-02")
}

func GenerateAuthor(name, email string) *Author {
	timeString := generateTime(time.Now())

	return &Author{
		Name:      name,
		Email:     email,
		CreatedAt: timeString,
	}
}

func (a *Author) Parse(r io.Reader, c *Commit) error {
	binary.Read(r, binary.BigEndian, a)
	return nil
}
