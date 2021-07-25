package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	q := GeneratePriorityQueue()
	q.Push(&Item{
		Value:    2,
		Priority: 2,
	})
	q.Push(&Item{
		Value:    1,
		Priority: 1,
	})
	q.Push(&Item{
		Value:    3,
		Priority: 3,
	})

	for _, s := range []int{3, 2, 1} {
		popvalue := q.Pop()
		v := popvalue.(int)
		assert.Equal(t, s, v)
	}

}
