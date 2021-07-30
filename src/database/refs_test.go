package database

import (
	"mygit/src/database/util"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AsciiControll(t *testing.T) {

	for _, d := range []struct {
		str      string
		expected bool
	}{
		{"\x00", true},
		{"\x01", true},
		{"\x02", true},
		{"\x03", true},
		{"\x04", true},
		{"\x05", true},
		{"\x06", true},
		{"\x07", true},
		{"\x08", true},
		{"\x09", true},
		{"\x0a", true},
		{"\x0b", true},
		{"\x0c", true},
		{"\x0e", true},
		{"\x0f", true},
		{"\x10", true},
		{"\x11", true},
		{"\x12", true},
		{"\x13", true},
		{"\x14", true},
		{"\x15", true},
		{"\x16", true},
		{"\x17", true},
		{"\x18", true},
		{"\x19", true},
		{"\x1a", true},
		{"\x1b", true},
		{"\x1c", true},
		{"\x1e", true},
		{"\x1f", true},
		{"*", true},
		{":", true},
		{"?", true},
		{`\[`, true},
		{"\\", true},
		{"^", true},
		{"~", true},
		{"\x7f", true},
		{"\x21", false},
		{"somedummy\x1esomedummy", true},
		{"\x1fsomesummy", true},
		{"somedummy*", true},
	} {
		t.Run("checkAscii", func(t *testing.T) {
			b := util.CheckRegExp(`[\x00-\x20*:?\[\\-~\x7f]`, d.str)
			assert.Equal(t, d.expected, b)
		})
	}
}
