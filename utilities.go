package utilities

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/invertedv/chutils"
	"github.com/invertedv/keyval"
)

// ***************  Search

// Position returns the index of needle in the haystack. It returns -1 if needle is not found.
// If haystack has length 1, then it is split into a slice with delimiter ",".
func Position(needle string, haystack ...string) int {
	var haySlice []string
	haySlice = haystack

	if len(haystack) == 1 {
		haySlice = strings.Split(haystack[0], ",")
	}

	for ind, straw := range haySlice {
		if straw == needle {
			return ind
		}
	}

	return -1
}

// Has returns true if needle is in haystack
func Has(needle string, haystack ...string) bool {
	return Position(needle, haystack...) >= 0
}

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

// ***************  Math

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
	return Slash(os.TempDir()) + "tmp" + randomLetters(length) + "." + ext
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
	return tmpDB + ".tmp" + randomLetters(length)
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
	qry := fmt.Sprintf("DROP TABLE %s", table)
	_, err := conn.Exec(qry)

	return err
}

// ***************  Misc

// randomLetters generates a string of length "length" by randomly choosing from a-z
func randomLetters(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"

	name := ""
	for ind := 0; ind < length; ind++ {

		//		randN := rand.Intn(len(letters))
		//		name += letters[randN : randN+1]
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
