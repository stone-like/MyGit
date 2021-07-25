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

func (a *Author) GetUnixTime() time.Time {
	t := time.Unix(int64(a.GetUnixTimeInt()), 0)

	return t
}

func (a *Author) GetUnixTimeInt() int {
	words := strings.Fields(a.CreatedAt)
	ut, _ := strconv.Atoi(words[0])
	return ut
}

func (a *Author) ShortTime() string {
	return a.GetUnixTime().Format("2006-01-02")
}

func (a *Author) ReadableTime() string {
	return a.GetUnixTime().Format("Mon Jan 2 15:4:5 2006 -0700")
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
