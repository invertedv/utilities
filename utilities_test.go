package utilities

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/invertedv/keyval"

	"github.com/stretchr/testify/assert"

	"gonum.org/v1/gonum/stat"
)

func TestRandInt(t *testing.T) {
	const (
		upper  = 10
		sample = 500000
	)

	p := 1.0 / float64(upper)
	sig := math.Sqrt(p * (1.0 - p) / float64(sample))

	start := time.Now()

	x, e := RandUnifInt(sample, upper)
	assert.Nil(t, e)
	fmt.Println("elapsed: ", time.Since(start).Seconds())
	xCnts := make([]int64, upper)
	for _, xval := range x {
		xCnts[xval]++
	}

	avgs := make([]float64, upper)
	for ind, xc := range xCnts {
		avgs[ind] = float64(xc) / sample
	}

	// check if these look uniform (beware multiple comparisons...)
	lowerCI := p - 3*sig
	upperCI := p + 3*sig
	for ind := 0; ind < upper; ind++ {
		assert.Less(t, avgs[ind], upperCI)
		assert.Greater(t, avgs[ind], lowerCI)
	}
}

func TestRandFlt(t *testing.T) {
	const (
		sample   = 3000000
		nBuckets = 20
	)

	p := 1.0 / nBuckets
	sig := math.Sqrt(p * (1.0 - p) / float64(sample))

	xs, e := RandUnifFlt(sample)
	assert.Nil(t, e)

	buckets := make([]int, nBuckets)
	for _, x := range xs {
		ind := MinInt(int(nBuckets*x), nBuckets-1)
		buckets[ind]++
	}

	avgs := make([]float64, nBuckets)
	for ind, buck := range buckets {
		avgs[ind] = float64(buck) / sample
	}

	// check if these look uniform
	lowerCI := p - 3*sig
	upperCI := p + 3*sig
	for ind := 0; ind < nBuckets; ind++ {
		assert.Less(t, avgs[ind], upperCI)
		assert.Greater(t, avgs[ind], lowerCI)
	}
}

func TestRandNorm(t *testing.T) {
	const (
		sample = 300000
	)

	xs, e := RandNorm(sample)
	assert.Nil(t, e)

	xMean := stat.Mean(xs, nil)
	xStd := stat.StdDev(xs, nil)

	// have a look at the mean
	z := xMean / (xStd / math.Sqrt(float64(sample)))
	assert.Less(t, z, 2.57)
	assert.Greater(t, z, -2.57)

}
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
