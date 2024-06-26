package utilities

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"
	"io"
	"io/fs"
	"math"
	"math/big"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/invertedv/chutils"
	f "github.com/invertedv/chutils/file"
	s "github.com/invertedv/chutils/sql"
	"github.com/invertedv/keyval"
)

// ***************  Search

// Position returns the index of needle in the haystack. It returns -1 if needle is not found.
// If haystack has length 1, then it is split into a slice with delimiter ",".
func Position(needle, delim string, haystack ...string) int {
	var haySlice []string
	haySlice = haystack

	if len(haystack) == 1 && delim != "" && strings.Contains(haystack[0], delim) {
		haySlice = strings.Split(haystack[0], delim)
	}

	for ind, straw := range haySlice {
		if straw == needle {
			return ind
		}
	}

	return -1
}

// Has returns true if needle is in haystack
func Has(needle, delim string, haystack ...string) bool {
	return Position(needle, delim, haystack...) >= 0
}

// DeDupeSlice removes duplicate entries of inSlc, sorting if sortReturn
func DeDupeSlice(inSlc []string, sortReturn bool) (outSlc []string) {
	for _, val := range inSlc {
		if !Has(val, ",", outSlc...) {
			outSlc = append(outSlc, val)
		}
	}

	if sortReturn {
		sort.Strings(outSlc)
	}

	return outSlc
}

// Matched returns a substring that is between the outermost set of startChar/endChar
func Matched(inStr, startChar, endChar string) (string, error) {
	start, ignore := -1, 0

	for ind := 0; ind < len(inStr); ind++ {
		switch inStr[ind : ind+1] {
		case startChar:
			if start == -1 {
				start = ind
			}
			ignore++
		case endChar:
			ignore--
			if ignore == 0 {
				return inStr[start+1 : ind], nil
			}
		}
	}

	if start == -1 {
		return "", nil
	}

	return "", fmt.Errorf("unmatched startChar")
}

// YesNo determines if inStr is yes/no, return true if "yes"
func YesNo(inStr string) (bool, error) {
	if inStr != "yes" && inStr != "no" && inStr != "" {
		return false, fmt.Errorf("expected yes/no, got %s", inStr)
	}

	if inStr == "no" || inStr == "" {
		return false, nil
	}

	return true, nil
}

// ***************  Math

// MaxInt returns the maximum of ints
func MaxInt(ints ...int) int {
	max := ints[0]
	for _, i := range ints {
		if i > max {
			max = i
		}
	}

	return max
}

// MinInt returns the minimum of ints
func MinInt(ints ...int) int {
	min := ints[0]
	for _, i := range ints {
		if i < min {
			min = i
		}
	}

	return min
}

// RandUnifInt generates a slice whose elements are random U[0,upper) int64's
func RandUnifInt(n, upper int) ([]int64, error) {
	const bytesPerInt = 8

	// generate random bytes
	b1 := make([]byte, bytesPerInt*n)
	if _, e := rand.Read(b1); e != nil {
		return nil, e
	}

	outInts := make([]int64, n)
	rdr := bytes.NewReader(b1)

	for ind := 0; ind < n; ind++ {
		r, e := rand.Int(rdr, big.NewInt(int64(upper)))
		if e != nil {
			return nil, e
		}
		outInts[ind] = r.Int64()
	}

	return outInts, nil
}

// RandUnifFlt generates a slice whose elements are random U(0, 1) floats
func RandUnifFlt(n int) ([]float64, error) {
	xs, e := RandUnifInt(n, math.MaxInt64)
	if e != nil {
		return nil, e
	}

	fltMax := float64(math.MaxInt64)
	us := make([]float64, n)

	for ind, x := range xs {
		us[ind] = float64(x) / fltMax
	}

	return us, nil
}

// RandNorm generates a slice whose elements are N(0,1)
func RandNorm(n int) ([]float64, error) {
	// algorithm generates normals in pairs
	nUnif := n + n%2

	xUnif, err := RandUnifFlt(nUnif)
	if err != nil {
		return nil, err
	}

	xNorm := make([]float64, n)

	for ind := 0; ind < n; ind += 2 {
		lnPart := math.Sqrt(-2.0 * math.Log(xUnif[ind]))
		angle := 2.0 * math.Pi * xUnif[ind+1]
		xNorm[ind] = lnPart * math.Cos(angle)
		if ind+1 < n {
			xNorm[ind+1] = lnPart * math.Sin(angle)
		}
	}

	return xNorm, nil
}

// ***************  Files

// TempFile produces a random temp file name in the system's tmp location.
// The file has extension "ext". The file name begins with "tmp" has length 3 + length.
func TempFile(ext string, length int) string {
	return Slash(os.TempDir()) + "tmp" + RandomLetters(length) + "." + ext
}

// ToFile writes string to file fileName, which is created
func ToFile(fileName, text string) error {
	handle, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer func() { _ = handle.Close() }()

	_, err = handle.WriteString(text)

	return err
}

// FileExists returns an error if "file" does not exist.
func FileExists(file string) error {
	// see if it is already there
	_, err := os.Stat(file)
	exists := !errors.Is(err, fs.ErrNotExist)

	if !exists {
		return fmt.Errorf("%s does not exist", file)
	}

	return nil
}

// CopyFile copies sourceFile to destFile
func CopyFile(sourceFile, destFile string) error {
	inFile, e := os.Open(sourceFile)
	if e != nil {
		return e
	}
	defer func() { _ = inFile.Close() }()

	outFile, e := os.Create(destFile)
	if e != nil {
		return e
	}
	defer func() { _ = outFile.Close() }()

	_, e = io.Copy(outFile, inFile)

	return e
}

// CopyFiles recursively copies files from fromDir to toDir
func CopyFiles(fromDir, toDir string) error {
	fromDir = Slash(fromDir)
	toDir = Slash(toDir)

	dirList, e := os.ReadDir(fromDir)
	if e != nil {
		return e
	}

	// skip if directory is empty
	if len(dirList) == 0 {
		return nil
	}

	if e := os.MkdirAll(toDir, os.ModePerm); e != nil {
		return e
	}

	for _, file := range dirList {
		if file.IsDir() {
			if e := CopyFiles(fromDir+file.Name(), toDir+file.Name()); e != nil {
				return e
			}
			continue
		}

		if e := CopyFile(fmt.Sprintf("%s%s", fromDir, file.Name()), fmt.Sprintf("%s%s", toDir, file.Name())); e != nil {
			return e
		}
	}

	return nil
}

// ***************  DB

// TempTable produces a random table name. The table name begins with "tmp".
// The table's name has length 3 +length.
// tmpDB is the database name.
func TempTable(tmpDB string, length int) string {
	return tmpDB + ".tmp" + RandomLetters(length)
}

// TableOrQuery takes table and returns a query. If it is already a query (has "select"), just returns the original value.
func TableOrQuery(table string) string {
	switch {
	case strings.Contains(strings.ToLower(table), "select"):
		// make sure it's in parens
		if table[0:1] != "(" {
			table = fmt.Sprintf("(%s)", table)
		}

		return table
	default:
		return fmt.Sprintf("(SELECT * FROM %s)", table)
	}
}

// DBExists returns an error if db does not exist
func DBExists(db string, conn *chutils.Connect) error {
	qry := fmt.Sprintf("EXISTS DATABASE %s", db)

	res, e := conn.Query(qry)
	if e != nil {
		return e
	}
	defer func() { _ = res.Close() }()

	var exist uint8
	res.Next()
	if e := res.Scan(&exist); e != nil {
		return e
	}

	if exist == 0 {
		return fmt.Errorf("db %s does not exist", db)
	}

	return nil
}

// TableExists returns an error if "table" does not exist.
// conn is the DB connector.
func TableExists(table string, conn *chutils.Connect) error {
	qry := fmt.Sprintf("SELECT * FROM %s LIMIT 1", TableOrQuery(table))
	_, err := conn.Exec(qry)

	if err != nil {
		return fmt.Errorf("table %s does not exist", table)
	}

	return nil
}

// BuildQuery replaces the placeholders with values
// placeholders have the form "?key".
// BuildQuery prepends a "?" to the keys in replacers.
func BuildQuery(srcQry string, replacers keyval.KeyVal) (qry string) {
	qry = srcQry
	for k, v := range replacers {
		qry = strings.ReplaceAll(qry, "?"+k, v.AsString)
	}

	return qry
}

// DropTable drops the table from ClickHouse
func DropTable(table string, conn *chutils.Connect) error {
	qry := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
	_, err := conn.Exec(qry)

	return err
}

// MakeConnection establishes the connection to ClickHouse, supplying common options
// - host, user, password are what you think
// - maxMemory: maximum memory (bytes) for a query
// - maxGroupBy: maximum memory to use in a GROUP BY
// - maxThreads: maximum # of threads for a query
func MakeConnection(host, user, password string, maxMemory, maxGroupBy int64, maxThreads int) (conn *chutils.Connect, err error) {
	if user == "" {
		user = GetTTYecho("ClickHouse User: ")
	}

	if password == "" {
		password = GetTTYnoecho("Clickhouse Password: ")
	}

	if conn, err = chutils.NewConnect(host, user, password, clickhouse.Settings{
		"max_memory_usage":                   maxMemory,
		"max_bytes_before_external_group_by": maxGroupBy,
		"max_threads":                        maxThreads,
	}); err != nil {
		return nil, err
	}

	return conn, nil
}

// QueryToCSV writes the output of the query to a CSV.
// - qry: query to run
// - csvFile: output file
// - quoteStings: if true, places strings in double quotes
// - header: if true, include header row of field names
// - conn: ClickHouse connection
func QueryToCSV(qry, csvFile string, quoteStrings, header bool, conn *chutils.Connect) error {
	handle, e := os.Create(csvFile)
	if e != nil {
		return e
	}
	defer func() { _ = handle.Close() }()

	rdr := s.NewReader(qry, conn)
	defer func() { _ = rdr.Close() }()

	if ex := rdr.Init("", chutils.MergeTree); ex != nil {
		return ex
	}

	if header {
		if _, e := handle.WriteString(strings.Join(rdr.TableSpec().FieldList(), ",") + "\n"); e != nil {
			return e
		}
	}

	quote := '"'
	if !quoteStrings {
		quote = 0
	}

	wtr := f.NewWriter(handle, csvFile, nil, ',', '\n', quote, "")

	// after = -1 means will not also produce a ClickHouse table
	ex := chutils.Export(rdr, wtr, -1, true)

	return ex
}

// GetTTYecho reads a response from the TTY while echoing the user's typing
func GetTTYecho(prompt string) string {
	rdr := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	txt, _ := rdr.ReadString('\n')
	return strings.ReplaceAll(txt, "\n", "")
}

// GetTTYnoecho reads a response from the TTYgit  without echoing the user's typing
func GetTTYnoecho(prompt string) string {
	fmt.Print(prompt)
	pass, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(pass)
}

// ***************  Misc

// ReplaceSmart replaces old with new except when it occurs within delim
func ReplaceSmart(source, oldChar, newChar, delim string) string {
	if len(oldChar) > 1 || len(newChar) > 1 || len(delim) > 1 {
		panic(fmt.Errorf("old, new or delim has multiple characters in ReplaceSmart"))
	}
	inside := false
	replaced := ""
	for ind := 0; ind < len(source); ind++ {
		ch := source[ind : ind+1]
		switch ch {
		case delim:
			inside = !inside
		case oldChar:
			if !inside {
				ch = newChar
			}
		}
		replaced += ch
	}

	return replaced
}

// ToLastDay moves a date to the last day of the month
func ToLastDay(dt time.Time) (eom time.Time) {
	yr, mon := dt.Year(), dt.Month()
	mon++
	if mon == 13 {
		mon = 1
		yr++
	}

	eom = time.Date(yr, mon, 1, 0, 0, 0, 0, time.UTC)
	eom = eom.Add(-24 * time.Hour)

	return eom
}

// PrettyDur returns a run duration in a minutes/seconds format
func PrettyDur(startTime time.Time) string {
	const secsPmin = 60

	secs := int(time.Since(startTime).Seconds())

	if secs < secsPmin {
		return fmt.Sprintf("%d seconds", secs)
	}

	mins := secs / 60
	secs -= mins * 60

	return fmt.Sprintf("%d minutes %d seconds", mins, secs)
}

// RandomLetters generates a string of length "length" by randomly choosing from a-z
func RandomLetters(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"

	randN, err := RandUnifInt(len(letters), len(letters))
	if err != nil {
		panic(err)
	}

	name := ""
	for ind := 0; ind < length; ind++ {
		name += letters[randN[ind] : randN[ind]+1]
	}

	return name
}

// Slash adds a trailing slash if inStr doesn't end in a slash
func Slash(inStr string) string {
	if inStr[len(inStr)-1] == '/' {
		return inStr
	}

	return inStr + "/"
}

// Aligner returns a slice of strings suitable for printing the two input slices
func Aligner[ChLeft, ChRight any | int32 | int64 | float32 | float64 | string | time.Time](left []ChLeft, right []ChRight, pad int) []string {
	if left == nil || right == nil || len(left) != len(right) {
		return nil
	}

	maxLen := 0
	var leftStr, outStr []string
	for ind := 0; ind < len(left); ind++ {
		str := fmt.Sprintf("%v", left[ind])
		maxLen = MaxInt(maxLen, len(str))
		leftStr = append(leftStr, str)
	}

	for ind := 0; ind < len(left); ind++ {
		padding := strings.Repeat(" ", maxLen-len(leftStr[ind])+pad)
		str := fmt.Sprintf("%s%s%v", leftStr[ind], padding, right[ind])
		outStr = append(outStr, str)
	}

	return outStr
}

// ***************  Type Conversions

// GTAny compares xa > xb
func GTAny(xa, xb any) (truth bool, err error) {
	if xb == nil || xa == nil {
		return true, nil
	}

	//	if reflect.TypeOf(xa) != reflect.TypeOf(xb) {
	//		return false, fmt.Errorf("compared values must be of same type, got %v and %v", reflect.TypeOf(xa), reflect.TypeOf(xb))
	//	}

	switch x := xa.(type) {
	case string:
		x = strings.ReplaceAll(x, "'", "")
		s, ok := xb.(string)
		if !ok {
			return false, fmt.Errorf("cannot convert to string")
		}

		y := strings.ReplaceAll(s, "'", "")
		return x > y, nil
	case int32:
		yy, e := Any2Int32(xb)
		if e != nil {
			return false, e
		}

		return x > *yy, nil
	case int64:
		yy, e := Any2Int64(xb)
		if e != nil {
			return false, e
		}

		return x > *yy, nil
	case float32:
		yy, e := Any2Float32(xb)
		if e != nil {
			return false, e
		}

		return x > *yy, nil
	case float64:
		yy, e := Any2Float64(xb)
		if e != nil {
			return false, e
		}

		return x > *yy, nil
	case time.Time:
		t, ok := xb.(time.Time)
		if !ok {
			return false, fmt.Errorf("cannot convert to date")
		}

		return x.Sub(t) > 0, nil
	}

	return false, fmt.Errorf("unsupported comparison")
}

// LTAny returns x<y for select underlying types of "any"
func LTAny(x, y any) (bool, error) {
	switch xt := x.(type) {
	case float64:
		return xt < y.(float64), nil
	case float32:
		return xt < y.(float32), nil
	case int64:
		return xt < y.(int64), nil
	case int32:
		return xt < y.(int32), nil
	case string:
		return xt < y.(string), nil
	case time.Time:
		return y.(time.Time).Sub(xt) > 0, nil
	default:
		return false, fmt.Errorf("cannot compare: LTAny")
	}
}

// Comparer compares xa and xb. Comparisons available are: ==, !=, >, <, >=, <=
func Comparer(xa, xb any, comp string) (truth bool, err error) {
	// a constant date comes in as a string
	if t1, e := Any2Date(xa); e == nil {
		xa = *t1
	}

	if t2, e := Any2Date(xb); e == nil {
		xb = *t2
	}

	test1, e1 := GTAny(xa, xb)
	if e1 != nil {
		return false, e1
	}

	test2, e2 := GTAny(xb, xa)
	if e2 != nil {
		return false, e2
	}

	switch comp {
	case ">":
		return test1, nil
	case ">=":
		return !test2, nil
	case "==":
		return !test1 && !test2, nil
	case "!=":
		return test1 || test2, nil
	case "<":
		return test2, nil
	case "<=":
		return !test1, nil
	}

	return false, fmt.Errorf("unsupported comparison: %s", comp)
}

// Any2Date attempts to convert inVal to a date (time.Time). Returns nil if this fails.
func Any2Date(inVal any) (*time.Time, error) {
	switch x := inVal.(type) {
	case string:
		formats := []string{"20060102", "1/2/2006", "01/02/2006", "Jan 2, 2006", "January 2, 2006", "Jan 2 2006", "January 2 2006"}
		for _, fmtx := range formats {
			dt, e := time.Parse(fmtx, strings.ReplaceAll(x, "'", ""))
			if e == nil {
				return &dt, nil
			}
		}
	case time.Time:
		return &x, nil
	case int, int32, int64:
		return Any2Date(fmt.Sprintf("%d", x))
	}

	return nil, fmt.Errorf("cannot convert %v to date: Any2Date", inVal)
}

// Any2Float64 attempts to convert inVal to float64.  Returns nil if this fails.
func Any2Float64(inVal any) (*float64, error) {
	var outVal float64

	switch x := inVal.(type) {
	case int:
		outVal = float64(x)
	case int32:
		outVal = float64(x)
	case int64:
		outVal = float64(x)
	case float32:
		outVal = float64(x)
	case float64:
		outVal = x
	case string:
		xx, e := strconv.ParseFloat(x, 64)
		if e != nil {
			return nil, e
		}
		outVal = xx
	default:
		return nil, fmt.Errorf("cannot convert %v to float64: Any2Float64", inVal)
	}

	return &outVal, nil
}

// Any2Float32 attempts to convert inVal to float32.  Returns nil if this fails.
func Any2Float32(inVal any) (*float32, error) {
	var outVal float32

	switch x := inVal.(type) {
	case int:
		outVal = float32(x)
	case int32:
		outVal = float32(x)
	case int64:
		outVal = float32(x)
	case float32:
		outVal = x
	case float64:
		outVal = float32(x)
	case string:
		xx, e := strconv.ParseFloat(x, 32)
		if e != nil {
			return nil, fmt.Errorf("cannot convert %v to float32: Any2Float32", inVal)
		}
		outVal = float32(xx)
	default:
		return nil, fmt.Errorf("cannot convert %v to float32: Any2Float32", inVal)
	}

	return &outVal, nil
}

// Any2Int64 attempts to convert inVal to int64.  Returns nil if this fails.
func Any2Int64(inVal any) (*int64, error) {
	var outVal int64

	switch x := inVal.(type) {
	case int:
		outVal = int64(x)
	case int32:
		outVal = int64(x)
	case int64:
		outVal = x
	case float32:
		if x > math.MaxInt64 || x < math.MinInt64 {
			return nil, fmt.Errorf("float32 out of range: Any2Int64")
		}

		outVal = int64(x)
	case float64:
		if x > math.MaxInt64 || x < math.MinInt64 {
			return nil, fmt.Errorf("float64 out of range: Any2Int64")
		}

		outVal = int64(x)
	case string:
		xx, e := strconv.ParseInt(x, 10, 64)
		if e != nil {
			return nil, fmt.Errorf("cannot convert %v to int64: Any2Int64", inVal)
		}

		outVal = xx
	default:
		return nil, fmt.Errorf("cannot convert %v to int64: Any2Int64", inVal)
	}

	return &outVal, nil
}

// Any2Int32 attempts to convert inVal to int32.  Returns nil if this fails.
func Any2Int32(inVal any) (*int32, error) {
	var outVal int32
	switch x := inVal.(type) {
	case int:
		outVal = int32(x)
	case int32:
		outVal = x
	case int64:
		if x > math.MaxInt32 || x < math.MinInt32 {
			return nil, fmt.Errorf("int out of range: Any2Int32")
		}

		outVal = int32(x)
	case float32:
		if x > math.MaxInt32 || x < math.MinInt32 {
			return nil, fmt.Errorf("float32 out of range: Any2Int32")
		}

		outVal = int32(x)
	case float64:
		if x > math.MaxInt32 || x < math.MinInt32 {
			return nil, fmt.Errorf("float64 out of range: Any2Int32")
		}

		outVal = int32(x)
	case string:
		xx, e := strconv.ParseInt(x, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("cannot convert %v to int32: Any2Int32", inVal)
		}

		outVal = int32(xx)
	default:
		return nil, fmt.Errorf("cannot convert %v to int32: Any2Int32", inVal)
	}

	return &outVal, nil
}

// Any2Int attempts to convert inVal to int.  Returns nil if this fails.
func Any2Int(inVal any) (*int, error) {
	var outVal int
	switch x := inVal.(type) {
	case int:
		outVal = x
	case int32:
		outVal = int(x)
	case int64:
		if x > math.MaxInt || x < math.MinInt {
			return nil, fmt.Errorf("int64 out of range: Any2Int")
		}

		outVal = int(x)
	case float32:
		if x > math.MaxInt || x < math.MinInt {
			return nil, fmt.Errorf("float32 out of range: Any2Int")
		}

		outVal = int(x)
	case float64:
		if x > math.MaxInt || x < math.MinInt {
			return nil, fmt.Errorf("float64 out of range: Any2Int")
		}

		outVal = int(x)
	case string:
		xx, e := strconv.ParseInt(x, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("cannot convert %v to int: Any2Int", inVal)
		}
		outVal = int(xx)
	default:
		return nil, fmt.Errorf("cannot convert %v to int: Any2Int", inVal)
	}

	return &outVal, nil
}

func Any2String(inVal any) string {
	switch x := inVal.(type) {
	case string:
		return x
	case time.Time:
		return x.Format("1/2/2006")
	case float32, float64:
		return fmt.Sprintf("%0.2f", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func AnySlice2Float64(inVal []any) ([]float64, error) {
	outVal := make([]float64, len(inVal))
	for ind, x := range inVal {
		xf, e := Any2Float64(x)
		if e != nil {
			return nil, e
		}
		outVal[ind] = *xf
	}

	return outVal, nil
}

func Any2Kind(inVal any, kind reflect.Kind) (any, error) {
	if inVal == nil {
		return nil, fmt.Errorf("input is nil: Any2Kind")
	}

	switch kind {
	case reflect.Float64:
		if xF64, e := Any2Float64(inVal); e == nil {
			return *xF64, nil
		}
	case reflect.Float32:
		if xF32, e := Any2Float32(inVal); e == nil {
			return *xF32, nil
		}
	case reflect.Int64:
		if xI64, e := Any2Int64(inVal); e == nil {
			return *xI64, nil
		}
	case reflect.Int32:
		if xI32, e := Any2Int32(inVal); e == nil {
			return *xI32, nil
		}
	case reflect.Int:
		if xI, e := Any2Int(inVal); e == nil {
			return *xI, nil
		}
	case reflect.String:
		return Any2String(inVal), nil
	case reflect.Struct:
		if xT, e := Any2Date(inVal); e == nil {
			return *xT, nil
		}
	}

	return nil, fmt.Errorf("unsupported type or conversion error: Any2Kind: %v", kind)
}

// String2Kind converts a string specifying a type to the reflect.Kind
func String2Kind(str string) reflect.Kind {
	switch str {
	case "float64":
		return reflect.Float64
	case "float32":
		return reflect.Float32
	case "string":
		return reflect.String
	case "int":
		return reflect.Int
	case "int32":
		return reflect.Int32
	case "int64":
		return reflect.Int64
	case "time.Time":
		return reflect.Struct
	default:
		return reflect.Interface
	}
}

// PrettyString returns a string version of x suitable for printing
func PrettyString(x any) string {
	// for determining # of places after the decimal
	const (
		t1 = 0.1
		t2 = 1.0
		t3 = 10.0
	)

	if x == nil {
		return ""
	}

	switch val := x.(type) {
	case int:
		return humanize.Comma(int64(val))
	case int32:
		return humanize.Comma(int64(val))
	case int64:
		return humanize.Comma(val)
	case float64:
		r := math.Abs(val)
		switch {
		case r < t1:
			return fmt.Sprintf("%0.4f", val)
		case r < t2:
			return fmt.Sprintf("%0.3f", val)
		case r < t3:
			return fmt.Sprintf("%0.2f", val)
		default:
			return fmt.Sprintf("%0.1f", val)
		}
	case float32:
		return PrettyString(float64(val))
	case string:
		return val
	case time.Time:
		return val.Format("2006-01-02")
	default:
		return ""
	}
}

// ToClickHouse returns a string suitable for a ClickHouse constant value
func ToClickHouse(inVal any) string {
	switch x := inVal.(type) {
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", inVal)
	case string:
		return fmt.Sprintf("'%s'", inVal)
	case time.Time:
		return fmt.Sprintf("'%s'", x.Format("20060102"))
	}

	return ""
}
