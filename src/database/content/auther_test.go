package content

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTimeString(t *testing.T) {
	ti := time.Now()

	ret := generateTime(ti)

	expected := fmt.Sprintf("%d %s", ti.Unix(), "+0900")

	assert.Equal(t, expected, ret)
}

func TestGetShortTime(t *testing.T) {
	ti := time.Now()
	c := fmt.Sprintf("%d %s", ti.Unix(), "+0900")
	a := &Author{
		CreatedAt: c,
	}

	st := a.ShortTime()

	assert.Equal(t, ti.Format("2006-01-02"), st)
}

func TestGetReadableTime(t *testing.T) {
	ti := time.Now()
	c := fmt.Sprintf("%d %s", ti.Unix(), "+0900")
	a := &Author{
		CreatedAt: c,
	}

	st := a.ReadableTime()

	assert.Equal(t, ti.Format("Mon Jan 2 15:4:5 2006 -0700"), st)
}
