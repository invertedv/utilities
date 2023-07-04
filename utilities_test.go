package utilities

import (
	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"math"
	"testing"
	"time"

	"github.com/invertedv/keyval"

	"github.com/stretchr/testify/assert"

	"gonum.org/v1/gonum/stat"
)

func TestReplaceSmart(t *testing.T) {
	inp := " ' x' "
	exp := "' x'"
	act := ReplaceSmart(inp, " ", "", "'")
	assert.Equal(t, exp, act)
}

func TestToLastDay(t *testing.T) {
	dts := []time.Time{
		time.Date(2008, 12, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2008, 2, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 2, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 3, 31, 0, 0, 0, 0, time.UTC),
	}

	exp := []time.Time{
		time.Date(2008, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2008, 2, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 2, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 3, 31, 0, 0, 0, 0, time.UTC),
	}

	for ind, dt := range dts {
		assert.Equal(t, exp[ind], ToLastDay(dt))
	}
}

func TestRandInt(t *testing.T) {
	const (
		upper  = 10
		sample = 500000
	)

	p := 1.0 / float64(upper)
	sig := math.Sqrt(p * (1.0 - p) / float64(sample))
	x, e := RandUnifInt(sample, upper)
	assert.Nil(t, e)

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
		loc := Position(in, "", haystack...)
		assert.Equal(t, loc, exp[ind])

		exp := loc >= 0
		has := Has(in, "", haystack...)
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

func TestRandomLetters(t *testing.T) {
	ltrs := RandomLetters(5)
	assert.Equal(t, len(ltrs), 5)
}

func TestAny2Date(t *testing.T) {
	inVals := []any{"20101101", 20200321, "feb 28, 1999", "October 3, 2001"}

	exp := []time.Time{
		time.Date(2010, 11, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 3, 21, 0, 0, 0, 0, time.UTC),
		time.Date(1999, 02, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2001, 10, 3, 0, 0, 0, 0, time.UTC),
	}

	for ind, inVal := range inVals {
		dt, e := Any2Date(inVal)
		assert.Nil(t, e)
		assert.Equal(t, exp[ind], *dt)
	}

	inVals = []any{"20010001", 3399, "feb 30, 2000"}
	for _, inVal := range inVals {
		_, e := Any2Date(inVal)
		assert.NotNil(t, e)
	}
}

func TestAny2Float32(t *testing.T) {
	inVals := []any{"3.14", 2.768, 3}
	exp := []float32{3.14, 2.768, 3}

	for ind, inVal := range inVals {
		x, e := Any2Float32(inVal)
		assert.Nil(t, e)
		assert.Equal(t, exp[ind], *x)
	}

	inVals = []any{time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC), "hello"}
	for _, inVal := range inVals {
		_, e := Any2Float32(inVal)
		assert.NotNil(t, e)
	}
}

func TestAny2Int32(t *testing.T) {
	inVals := []any{"3", 22, 3}
	exp := []int32{3, 22, 3}

	for ind, inVal := range inVals {
		x, e := Any2Int32(inVal)
		assert.Nil(t, e)
		assert.Equal(t, exp[ind], *x)
	}

	inVals = []any{time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
		"hello",
		int64(math.MaxInt64),
		float32(math.MaxInt64),
		float64(math.MaxInt64)}

	for _, inVal := range inVals {
		_, e := Any2Int32(inVal)
		assert.NotNil(t, e)
	}
}

func TestMatched(t *testing.T) {
	inStrs := []string{"a(bb(cc) dd (ee))", "a[b[c[d]]]", "abc"}
	results := []string{"bb(cc) dd (ee)", "b[c[d]]", ""}
	seps := []string{"()", "[]", "()"}

	for ind, inStr := range inStrs {
		subStr, e := Matched(inStr, seps[ind][0:1], seps[ind][1:2])
		assert.Nil(t, e)
		assert.Equal(t, results[ind], subStr)
	}

	inStrs = []string{"a(b()"}
	seps = []string{"()"}

	for ind, inStr := range inStrs {
		_, e := Matched(inStr, seps[ind][0:1], seps[ind][1:2])
		assert.NotNil(t, e)
	}
}

func TestHTML2File(t *testing.T) {
	htmlFile := "/home/will/tmp/marginal.html"
	e := HTML2File(htmlFile, "png", "/home/will/tmp", "marginal")
	assert.Nil(t, e)
}

func TestFig2File(t *testing.T) {
	x := []string{"A", "B", "C", "D"}
	y := []float32{5, 6, 7, 8}
	histPlot := &grob.Bar{X: x, Y: y}
	fig := &grob.Fig{Data: grob.Traces{histPlot}}
	lay := &grob.Layout{Width: 800, Height: 600, Title: &grob.LayoutTitle{Text: "Bar Chart"}}
	fig.Layout = lay
	e := Fig2File(fig, "png", "/home/will/tmp", "testpng")
	assert.Nil(t, e)
}
