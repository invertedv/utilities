package utilities

import (
	"testing"

	"github.com/invertedv/keyval"

	"github.com/stretchr/testify/assert"
)

func TestMaxInt(t *testing.T) {
	ins := [][]int{
		{1, 2, 3, 4},
		{4, 3, 2, 1},
		{3, 1, 2, 4},
		{1},
		{1, 1, 1, 1},
	}

	expMax := []int{
		4,
		4,
		4,
		1,
		1,
	}

	expMin := []int{
		1,
		1,
		1,
		1,
		1,
	}

	for ind, in := range ins {
		max := MaxInt(in...)
		assert.Equal(t, expMax[ind], max)

		min := MinInt(in...)
		assert.Equal(t, expMin[ind], min)
	}

	assert.Equal(t, 1, MinInt(1, 2, 3))
	assert.Equal(t, 10, MaxInt(1, 10, 8))
}

func TestPosition(t *testing.T) {
	haystack := []string{"a", "b", "c", "d"}
	ins := []string{
		"c",
		"a",
		"d",
		"g"}
	exp := []int{
		2,
		0,
		3,
		-1}

	for ind, in := range ins {
		loc := Position(in, haystack...)
		assert.Equal(t, loc, exp[ind])

		exp := loc >= 0
		has := Has(in, haystack...)
		assert.Equal(t, exp, has)
	}
}

func TestBuildQuery(t *testing.T) {
	qry := "SELECT ?field FROM ?table"
	repl := make(keyval.KeyVal)
	repl["field"] = keyval.Populate("xTest")
	repl["table"] = keyval.Populate("db.table")

	qryOut := BuildQuery(qry, repl)
	qryExp := "SELECT xTest FROM db.table"
	assert.Equal(t, qryExp, qryOut)
}
