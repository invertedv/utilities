package utilities

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/invertedv/chutils"
	"github.com/invertedv/keyval"
)

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

// Slash adds a trailing slash if inStr doesn't end in a slash
func Slash(inStr string) string {
	if inStr[len(inStr)-1] == '/' {
		return inStr
	}

	return inStr + "/"
}

// randomLetters generates a string of length "length" by randomly choosing from a-z
func randomLetters(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"

	name := ""
	for ind := 0; ind < length; ind++ {
		randN := rand.Intn(len(letters))
		name += letters[randN : randN+1]
	}

	return name
}

// TempTable produces a random table name. The table name begins with "tmp".
// The table's name has length 3 +length.
// tmpDB is the database name.
func TempTable(tmpDB string, length int) string {
	return tmpDB + ".tmp" + randomLetters(length)
}

// TempFile produces a random temp file name in the system's tmp location.
// The file has extension "ext". The file name begins with "tmp" has length 3 + length.
func TempFile(ext string, length int) string {
	return Slash(os.TempDir()) + "tmp" + randomLetters(length) + "." + ext
}

// FileExists returns an error if "file" does not exist.
func FileExists(file string) error {
	_, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("file %s does not exist", file)
	}

	return nil
}

// TableExists returns an error if "table" does not exist.
// conn is the DB connector.
func TableExists(table string, conn *chutils.Connect) error {
	qry := fmt.Sprintf("SELECT * FROM %s LIMIT 1", table)
	_, err := conn.Exec(qry)

	if err != nil {
		return fmt.Errorf("table %s does not exist", table)
	}

	return nil
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
