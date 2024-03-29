package utilities

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/MetalBlueberry/go-plotly/offline"
	"github.com/invertedv/chutils"
	s "github.com/invertedv/chutils/sql"
)

// nameLength is the length of random characters for name of temp files
const nameLength = 8

// Browser is the browser in which to show plots.  This starts as the system default, but can be changed.
// For instance, Browser="vivaldi" will place the plot in Vivaldi.
var Browser = "xdg-open"

// PlotlyImage is the type of image file to create
type PlotlyImage int

const (
	PlotlyJPEG PlotlyImage = 0 + iota
	PlotlyPNG
	PlotlyHTML
	PlotlyPDF
	PlotlyWEBP
	PlotlySVG
	PlotlyEPS
	PlotlyEMF
)

func (pi PlotlyImage) String() string {
	switch pi {
	case PlotlyJPEG:
		return "jpeg"
	case PlotlyPNG:
		return "png"
	case PlotlyHTML:
		return "html"
	case PlotlyPDF:
		return "pdf"
	case PlotlyWEBP:
		return "webp"
	case PlotlySVG:
		return "svg"
	case PlotlyEPS:
		return "eps"
	case PlotlyEMF:
		return "emf"
	}

	return ""
}

// PlotDef specifies Plotly Layout features I commonly use.
type PlotDef struct {
	Show       bool          // Show - true = show graph in browser
	Title      string        // Title - plot title
	XTitle     string        // XTitle - x-axis title
	YTitle     string        // Ytitle - y-axis title
	STitle     string        // STitle - sub-title (under the x-axis)
	Legend     bool          // Legend - true = show legend
	Height     float64       // Height - height of graph, in pixels
	Width      float64       // Width - width of graph, in pixels
	FileName   string        // FileName - output file for (no suffix, no path)
	OutDir     string        // Outdir - output directory
	ImageTypes []PlotlyImage // image type(s) to create (e.g. png, jpg...)
}

// Plotter plots the Plotly Figure fig with Layout lay.  The layout is augmented by
// features I commonly use.
//
//	fig      plotly figure
//	lay      plotly layout (nil is OK)
//	pd       PlotDef structure with plot options.
//
// lay can be initialized with any additional layout options needed.
func Plotter(fig *grob.Fig, lay *grob.Layout, pd *PlotDef) error {
	// convert newlines to <br>
	pd.Title = strings.ReplaceAll(pd.Title, "\n", "<br>")
	pd.STitle = strings.ReplaceAll(pd.STitle, "\n", "<br>")
	pd.XTitle = strings.ReplaceAll(pd.XTitle, "\n", "<br>")
	pd.YTitle = strings.ReplaceAll(pd.YTitle, "\n", "<br>")

	if lay == nil {
		lay = &grob.Layout{}
	}

	if pd.Title != "" {
		lay.Title = &grob.LayoutTitle{Text: pd.Title}
	}

	if pd.YTitle != "" {
		if lay.Yaxis == nil {
			lay.Yaxis = &grob.LayoutYaxis{Title: &grob.LayoutYaxisTitle{Text: pd.YTitle}}
		} else {
			lay.Yaxis.Title = &grob.LayoutYaxisTitle{Text: pd.YTitle}
		}
		lay.Yaxis.Showline = grob.True
	}

	if pd.XTitle != "" {
		xTitle := pd.XTitle
		if pd.STitle != "" {
			xTitle += fmt.Sprintf("<br>%s", pd.STitle)
		}

		if lay.Xaxis == nil {
			lay.Xaxis = &grob.LayoutXaxis{Title: &grob.LayoutXaxisTitle{Text: xTitle}}
		} else {
			lay.Xaxis.Title = &grob.LayoutXaxisTitle{Text: xTitle}
		}
	}

	if !pd.Legend {
		lay.Showlegend = grob.False
	}

	if pd.Width > 0.0 {
		lay.Width = pd.Width
	}

	if pd.Height > 0.0 {
		lay.Height = pd.Height
	}

	fig.Layout = lay

	// output to file(s)
	if pd.FileName != "" && pd.ImageTypes != nil {
		for _, ft := range pd.ImageTypes {
			outDir := fmt.Sprintf("%s%v", Slash(pd.OutDir), ft)
			// create it if it's not there
			if e := os.MkdirAll(outDir, os.ModePerm); e != nil {
				return e
			}

			if e := Fig2File(fig, ft, outDir, pd.FileName); e != nil {
				return e
			}
		}
	}

	if pd.Show {
		// create temp file.  We'll return this, in case it's needed
		pd.FileName = TempFile("html", nameLength)

		offline.ToHtml(fig, pd.FileName)

		var cmd *exec.Cmd
		if Browser != "xdg-open" {
			cmd = exec.Command(Browser, "-url", pd.FileName)
		} else {
			cmd = exec.Command(Browser, pd.FileName)
		}

		if e := cmd.Start(); e != nil {
			return e
		}

		time.Sleep(time.Second) // need to pause while browser loads graph

		if e := os.Remove(pd.FileName); e != nil {
			return e
		}
	}

	return nil
}

// Fig2File outputs a plotly figure to a graphics file (png, jpg, etc.)
// This func requires that orca be installed.
// Inputs:
//   - fig.  plotly figure
//   - plotType.  graph type. One of: png, jpeg, webp, svg, pdf, eps, emf
//   - outDir.  Output directory.
//   - outFile. Filename of output, with NO extension.
func Fig2File(fig *grob.Fig, plotType PlotlyImage, outDir, outFile string) error {
	if strings.Contains(outFile, ".") {
		return fmt.Errorf("no extension allowed for outFile in Fig2File")
	}

	if plotType < 0 || plotType > PlotlyEMF {
		return fmt.Errorf("illegal plotType in Fig2File. Values between 0 and 7")
	}

	if plotType == PlotlyHTML {
		fileName := fmt.Sprintf("%s%s.html", Slash(outDir), outFile)
		offline.ToHtml(fig, fileName)
		return nil
	}

	figBytes, err := json.Marshal(fig)
	figStr := string(figBytes)
	if err != nil {
		panic(err)
	}

	tempFileName := TempFile("js", nameLength)

	var tempFile *os.File
	if tempFile, err = os.Create(tempFileName); err != nil {
		return err
	}

	if _, e := tempFile.WriteString(figStr); e != nil {
		return e
	}

	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempFileName) }()

	comm := fmt.Sprintf("orca graph %s --no-sandbox -f %s -d %s  -o %s.%s", tempFileName, plotType, outDir, outFile, plotType)
	cmd := exec.Command("bash", "-c", comm)

	return cmd.Run()
}

// HTML2File produces an image file from a plotly html file
// This func requires that orca be installed.
// Inputs
//   - htmlFile.  plotly html file
//   - plotType.  graph type. One of: png, jpeg, webp, svg, pdf, eps, emf
//   - outDir.  Output directory.
//   - outFile. Filename of output, with NO extension.
func HTML2File(htmlFile string, plotType PlotlyImage, outDir, outFile string) error {
	var (
		handle *os.File
		err    error
	)

	if handle, err = os.Open(htmlFile); err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()

	var plot []byte
	if plot, err = io.ReadAll(handle); err != nil {
		return err
	}

	plotStr := string(plot)

	indx := strings.Index(plotStr, "JSON.parse")

	if indx < 0 {
		return fmt.Errorf("not a plotly html file: %s", htmlFile)
	}

	var jsonStr string
	if jsonStr, err = Matched(plotStr[indx+10:], "(", ")"); err != nil {
		return err
	}

	tempFileName := TempFile("js", nameLength)

	var tempFile *os.File
	if tempFile, err = os.Create(tempFileName); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tempFileName) }()

	if _, e := tempFile.WriteString(jsonStr[1 : len(jsonStr)-1]); e != nil {
		return e
	}

	_ = tempFile.Close()

	comm := fmt.Sprintf("orca graph %s --no-sandbox -f %s -d %s  -o %s.%s", tempFileName, plotType, outDir, outFile, plotType)
	cmd := exec.Command("bash", "-c", comm)

	return cmd.Run()
}

// HistData represents a histogram constructed from querying ClickHouse
type HistData struct {
	Levels   []any             // levels of the field
	Counts   []int64           // counts
	Prop     []float32         // proportions
	Total    int64             // total counts
	Qry      string            // query used to pull the data
	FieldDef *chutils.FieldDef // field defs of returns
	Fig      *grob.Fig         // histogram
}

// NewHistData pulls the data from ClickHouse and creates a plotly histogram
func NewHistData(rootQry, field, where string, conn *chutils.Connect) (*HistData, error) {
	hd := &HistData{Qry: rootQry}

	var qry string
	switch where == "" {
	case true:
		qry = fmt.Sprintf("WITH d AS (%s) SELECT %s, toInt64(COUNT(*)) AS n FROM d GROUP BY %s ORDER BY %s", rootQry, field, field, field)
	case false:
		qry = fmt.Sprintf("WITH d AS (%s) SELECT %s, toInt64(COUNT(*)) AS n FROM d WHERE %s GROUP BY %s ORDER BY %s", rootQry, field, where, field, field)
	}

	rdr := s.NewReader(qry, conn)
	defer func() { _ = rdr.Close() }()

	if e := rdr.Init("", chutils.MergeTree); e != nil {
		return nil, e
	}

	rows, _, e := rdr.Read(0, false)
	if e != nil {
		return nil, e
	}

	for ind := 0; ind < len(rows); ind++ {
		hd.Levels = append(hd.Levels, rows[ind][0])
		n := rows[ind][1].(int64)
		hd.Counts = append(hd.Counts, n)
		hd.Total += n
	}

	nFloat := float32(hd.Total)
	for ind := 0; ind < len(rows); ind++ {
		hd.Prop = append(hd.Prop, float32(hd.Counts[ind])/nFloat)
	}

	_, hd.FieldDef, _ = rdr.TableSpec().Get(field)
	histPlot := &grob.Bar{X: hd.Levels, Y: hd.Prop, Type: grob.TraceTypeBar}
	hd.Fig = &grob.Fig{Data: grob.Traces{histPlot}}

	return hd, nil
}

func (hd *HistData) String() string {
	return strings.Join(Aligner(hd.Levels, hd.Counts, 5), "\n")
}

// QuantileData represents a quantile plot constructed from querying ClickHouse
type QuantileData struct {
	Q        []float32         // quantiles at u
	U        []float32         // u values (0-1)
	Total    int64             // total sample size
	Qry      string            // query used to pull the data
	FieldDef *chutils.FieldDef // field defs of fields pulled
	Fig      *grob.Fig         // quantile plot
}

// NewQuantileData pulls the data from ClickHouse and creates a plotly quantile plot
func NewQuantileData(rootQry, field, where string, conn *chutils.Connect) (*QuantileData, error) {
	var (
		ptiles []string
	)

	outQ := &QuantileData{}

	for ind := 0; ind < 100; ind++ {
		u := float32(ind) / 100
		outQ.U = append(outQ.U, u)
		ptiles = append(ptiles, fmt.Sprintf("%v", u))
	}

	ptile := strings.Join(ptiles, ",")

	var qry, qryTot string
	switch where == "" {
	case true:
		qry = fmt.Sprintf("WITH d AS (%s) SELECT toFloat32(arrayJoin(quantiles(%s)(%s))) AS q FROM d", rootQry, ptile, field)
		qryTot = fmt.Sprintf("WITH d AS (%s) SELECT toInt64(COUNT(*)) AS n FROM d", rootQry)
	case false:
		qry = fmt.Sprintf("WITH d AS (%s) SELECT toFloat32(arrayJoin(quantiles(%s)(%s))) AS q FROM d WHERE %s", rootQry, ptile, field, where)
		qryTot = fmt.Sprintf("WITH d AS (%s) SELECT toInt64(COUNT(*)) AS n FROM d WHERE %s", rootQry, where)
	}

	outQ.Qry = qry

	rdr := s.NewReader(qryTot, conn)
	if e := rdr.Init("", chutils.MergeTree); e != nil {
		return nil, e
	}

	rows, _, e := rdr.Read(1, false)
	if e != nil {
		return nil, e
	}
	outQ.Total = rows[0][0].(int64)

	rdr = s.NewReader(qry, conn)
	defer func() { _ = rdr.Close() }()

	if ex := rdr.Init("", chutils.MergeTree); ex != nil {
		return nil, ex
	}
	_, outQ.FieldDef, _ = rdr.TableSpec().Get(field)

	rows, _, e = rdr.Read(0, false)
	if e != nil {
		return nil, e
	}

	for ind := 0; ind < len(rows); ind++ {
		outQ.Q = append(outQ.Q, rows[ind][0].(float32))
	}

	outQ.Fig = &grob.Fig{Data: grob.Traces{&grob.Scatter{X: outQ.U, Y: outQ.Q, Mode: grob.ScatterModeLines}}}

	return outQ, nil
}

type XYData struct {
	X         []any               // quantiles at u
	Y         [][]any             // u values (0-1)
	Qry       string              // query used to pull the data
	XfieldDef *chutils.FieldDef   // field def of X field
	YfieldDef []*chutils.FieldDef // field def of Y field
	Fig       *grob.Fig           // xy plot
}

func NewXYData(rootQry, where, fields, colors, lineTypes string, conn *chutils.Connect) (*XYData, error) {
	var err error
	outXY := &XYData{}
	outXY.Fig = &grob.Fig{}

	fieldsSlc := strings.Split(fields, ",")
	colorsSlc := strings.Split(colors, ",")
	lineTypeSlc := strings.Split(lineTypes, ",")

	if len(fieldsSlc) > 1 {
		if len(fieldsSlc) != len(colorsSlc)+1 || len(fieldsSlc) != len(lineTypeSlc)+1 {
			return nil, fmt.Errorf("lineTypes or colors not correct length")
		}
	}

	var qry string
	switch where == "" {
	case true:
		qry = rootQry
	case false:
		qry = fmt.Sprintf("WITH d AS (%s) SELECT * FROM d WHERE %s", rootQry, where)
	}

	outXY.Qry = qry

	rdr := s.NewReader(qry, conn)
	defer func() { _ = rdr.Close() }()

	if ex := rdr.Init("", chutils.MergeTree); ex != nil {
		return nil, ex
	}

	colX, startCol := 0, 0
	if len(fieldsSlc) > 1 {
		colX, outXY.XfieldDef, _ = rdr.TableSpec().Get(fieldsSlc[0])
		startCol = 1
	}

	rows, _, e := rdr.Read(0, false)
	if e != nil {
		return nil, e
	}

	for col := 0; col < len(fieldsSlc)-startCol; col++ {
		var thisY []any
		yField := strings.Trim(fieldsSlc[col+startCol], " ")
		var (
			colY    int
			fldDefY *chutils.FieldDef
		)

		if colY, fldDefY, err = rdr.TableSpec().Get(yField); err != nil {
			return nil, err
		}

		outXY.YfieldDef = append(outXY.YfieldDef, fldDefY)

		thisY = toSlice(rows, colY)
		if col == 0 {
			outXY.X = toSlice(rows, colX)
		}

		outXY.Y = append(outXY.Y, thisY)
		//TODO: color not getting to boxplot
		switch lineTypeSlc[col] == "box" {
		case false:
			var tr *grob.Scatter
			switch lineTypeSlc[col] {
			case "m":
				tr = &grob.Scatter{Name: fldDefY.Name, X: outXY.X, Y: thisY,
					Mode: grob.ScatterModeMarkers, Marker: &grob.ScatterMarker{Color: colorsSlc[col]}}
			case "l":
				tr = &grob.Scatter{Name: fldDefY.Name, X: outXY.X, Y: thisY,
					Mode: grob.ScatterModeLines, Line: &grob.ScatterLine{Color: colorsSlc[col]}}
			case "avg":
				x, y, _, _, _ := means(rootQry, where, outXY.XfieldDef.Name, fldDefY.Name, false, conn)
				tr = &grob.Scatter{Name: "mean " + fldDefY.Name, X: x, Y: y,
					Mode: grob.ScatterModeLines, Line: &grob.ScatterLine{Color: colorsSlc[col]}}
			case "median":
				x, _, y, _, _ := means(rootQry, where, outXY.XfieldDef.Name, fldDefY.Name, true, conn)
				tr = &grob.Scatter{Name: "median " + fldDefY.Name, X: x, Y: y,
					Mode: grob.ScatterModeLines, Line: &grob.ScatterLine{Color: colorsSlc[col]}}
			case "se":
				x, y, _, low, high := means(rootQry, where, outXY.XfieldDef.Name, fldDefY.Name, false, conn)
				tr = &grob.Scatter{Name: "mean " + fldDefY.Name, X: x, Y: y,
					Mode: grob.ScatterModeLines, Line: &grob.ScatterLine{Color: colorsSlc[col]}}
				whisker(x, low, high, colorsSlc[col], outXY.Fig)
			case "quartile":
				x, _, y, low, high := means(rootQry, where, outXY.XfieldDef.Name, fldDefY.Name, true, conn)
				tr = &grob.Scatter{Name: "median " + fldDefY.Name, X: x, Y: y,
					Mode: grob.ScatterModeLines, Line: &grob.ScatterLine{Color: colorsSlc[col]}}
				whisker(x, low, high, colorsSlc[col], outXY.Fig)
			default:
				return nil, fmt.Errorf("unknown line type: %s", lineTypeSlc[col])
			}
			outXY.Fig.AddTraces(tr)
		case true:
			tr := &grob.Box{Name: fldDefY.Name, X: outXY.X, Y: thisY, Type: grob.TraceTypeBox, Boxpoints: false,
				Marker: &grob.BoxMarker{Color: colorsSlc[col]}}
			outXY.Fig.AddTraces(tr)
		}
	}

	return outXY, nil
}

// toSlice pulls a column out of x
func toSlice(x []chutils.Row, col int) []any {
	var out []any
	for ind := 0; ind < len(x); ind++ {
		out = append(out, x[ind][col])
	}

	return out
}

// means returns the means, +/- 2 std dev,median & quartiles when the query groups by the xField.
func means(rootQry, where, xField, yField string, quartiles bool, conn *chutils.Connect) (avgX, avgY, medianY, low, high []any) {
	// skeleton query
	const blnkQry = `
WITH d AS (%s) 
  SELECT %s, 
  avg(%s) AS avg%s, 
  avg%s-2*stddevSamp(%s) AS lowSe,
  avg%s+2*stddevSamp(%s) AS upSe,
  quantile(.25)(%s) AS lq,
  quantile(.5)(%s) AS median,
  quantile(.75)(%s) AS uq
FROM d 
%s 
GROUP BY %s 
ORDER BY %s
`

	if where != "" {
		where = fmt.Sprintf("WHERE %s ", where)
	}

	// put flesh on the skeleton
	qry := fmt.Sprintf(blnkQry, rootQry, xField, yField, yField, yField,
		yField, yField, yField, yField, yField, yField, where, xField, xField)

	rdr := s.NewReader(qry, conn)
	defer func() { _ = rdr.Close() }()

	if ex := rdr.Init("", chutils.MergeTree); ex != nil {
		return nil, nil, nil, nil, nil
	}

	rows, _, e := rdr.Read(0, false)
	if e != nil {
		return nil, nil, nil, nil, nil
	}

	avgX = toSlice(rows, 0)
	avgY = toSlice(rows, 1)
	medianY = toSlice(rows, 5)
	if !quartiles {
		low = toSlice(rows, 2)
		high = toSlice(rows, 3)
	} else {
		low = toSlice(rows, 4)
		high = toSlice(rows, 6)
	}

	return avgX, avgY, medianY, low, high
}

// whisker adds whiskers (e.g. CIs) to Fig as represented by yLow & yHigh
func whisker(x, yLow, yHigh []any, color string, fig *grob.Fig) {
	var width float64
	switch x[0].(type) {
	case float32, float64:
		width = 1
	default:
		width = .25
	}

	for ind, xVal := range x {
		xs := []any{xVal, xVal}
		ys := []any{yLow[ind], yHigh[ind]}

		tr1 := &grob.Bar{X: xs, Y: ys, Marker: &grob.BarMarker{Color: color}, Showlegend: grob.False, Width: width, Type: grob.TraceTypeBar} //.25
		fig.AddTraces(tr1)
	}
}
