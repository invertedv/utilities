package utilities

import (
	"os"
	"strings"
	"time"
)

// Table holds a table
type Table struct {
	RowNames []string
	ColNames []string
	Data     [][]any // stored by columns
	markdown bool
}

// Write writes the table to a file.  If markDown a markdown table is created.
func (cd *Table) Write(outFile string, markDown bool) error {
	// write out file
	var (
		fl  *os.File
		err error
	)

	cd.markdown = markDown

	if fl, err = os.Create(outFile); err != nil {
		return err
	}
	defer func() { _ = fl.Close() }()

	if _, e := fl.WriteString(cd.String()); e != nil {
		return e
	}

	cd.markdown = false

	return nil
}

// CleanUp removes empty rows. A row is not empty if it has an element that is float/int/date or a string
// that are not ""
func (cd *Table) CleanUp() {
	var (
		rowNames []string
	)

	nCol := len(cd.Data)
	nRow := len(cd.Data[0])
	dataVal := make([][]any, nCol)

	for row := 0; row < nRow; row++ {
		ok := false
		for col := 0; col < nCol; col++ {
			switch x := cd.Data[col][row].(type) {
			case int, int32, int64, float32, float64, time.Time:
				ok = true
			case string:
				if x != "" {
					ok = true
				}
			}

			if ok {
				break
			}
		}
		// row is empty
		if ok {
			rowNames = append(rowNames, cd.RowNames[row])

			for col := 0; col < nCol; col++ {
				dataVal[col] = append(dataVal[col], cd.Data[col][row])
			}
		}
	}

	cd.RowNames = rowNames
	cd.Data = dataVal
}

// Pad pads each element of inTable so the columns line up with pad spaces separating them
func Pad(inTable [][]string, pad int) string {
	maxes := make([]int, len(inTable[0]))

	for row := 0; row < len(inTable); row++ {
		for col := 0; col < len(inTable[0]); col++ {
			inTable[row][col] = strings.Trim(inTable[row][col], " ") // get rid of these
			if l := len(inTable[row][col]); l > maxes[col] {
				maxes[col] = l
			}
		}
	}

	outString := ""
	for row := 0; row < len(inTable); row++ {
		for col := 0; col < len(inTable[0]); col++ {
			element := inTable[row][col] + strings.Repeat(" ", maxes[col]+pad-len(inTable[row][col]))
			outString += element
		}
		outString += "\n"
	}

	return outString
}

func (cd *Table) String() string {
	const padLength = 4

	if len(cd.Data) == 0 {
		return ""
	}

	var outSlc [][]string // stored by rows

	// if cd.markdown add pipes to create a Markdown table
	sep := ""
	if cd.markdown {
		sep = "|"
	}

	var colNames []string
	for ind := 0; ind < len(cd.ColNames); ind++ {
		colNames = append(colNames, cd.ColNames[ind]+sep)
	}
	outSlc = append(outSlc, colNames)

	for row := 0; row < len(cd.Data[0]); row++ {
		rowSlc := []string{cd.RowNames[row] + sep}
		for col := 0; col < len(cd.Data); col++ {
			rowSlc = append(rowSlc, PrettyString(cd.Data[col][row])+sep)
		}

		outSlc = append(outSlc, rowSlc)
	}

	return Pad(outSlc, padLength)
}
