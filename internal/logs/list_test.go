package logs

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvictingList_Values(t *testing.T) {
	l := NewEvictingList[int](5)
	l.Add(1)
	l.Add(2)
	l.Add(3)
	assert.Equal(t, []int{1, 2, 3}, l.Values())

	l.Add(4)
	l.Add(5)
	l.Add(6)
	l.Add(7)
	assert.Equal(t, []int{3, 4, 5, 6, 7}, l.Values())
}
