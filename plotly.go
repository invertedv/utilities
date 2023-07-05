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

// PlotlyImage is the type of image file to create
type PlotlyImage int

const (
	PlotlyJPG PlotlyImage = 0 + iota
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
	case PlotlyJPG:
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
	Show      bool          // Show - true = show graph in browser
	Title     string        // Title - plot title
	XTitle    string        // XTitle - x-axis title
	YTitle    string        // Ytitle - y-axis title
	STitle    string        // STitle - sub-title (under the x-axis)
	Legend    bool          // Legend - true = show legend
	Height    float64       // Height - height of graph, in pixels
	Width     float64       // Width - width of graph, in pixels
	FileName  string        // FileName - output file for (no suffix, no path)
	OutDir    string        // Outdir - output directory
	FileTypes []PlotlyImage // image type(s) to create (e.g. png, jpg...)
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
			lay.Xaxis.Title = &grob.LayoutXaxisTitle{Text: pd.YTitle}
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
	if pd.FileName != "" && pd.FileTypes != nil {
		for _, ft := range pd.FileTypes {
			if e := Fig2File(fig, ft, pd.OutDir, pd.FileName); e != nil {
				return e
			}
		}
	}

	if pd.Show {
		tmp := false
		if pd.FileName == "" {
			tmp = true
			// create temp file.  We'll return this, in case it's needed
			pd.FileName = TempFile("plotly", nameLength)
		}

		offline.ToHtml(fig, pd.FileName)
		cmd := exec.Command(Browser, "-url", pd.FileName)

		if e := cmd.Start(); e != nil {
			return e
		}
		time.Sleep(time.Second)

		if tmp {
			// need to pause while browser loads graph

			if e := os.Remove(pd.FileName); e != nil {
				return e
			}
		}
	}

	return nil
}

// Browser is the browser in which to show plots.  This is not set to the system default so the user has control...
var Browser = "firefox"

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

type HistData struct {
	Levels []any
	Counts []int32
}

func NewHistData(rootQry, field string, conn *chutils.Connect) (*HistData, error) {
	qry := fmt.Sprintf("WITH d AS (%s) SELECT %s, toInt32(COUNT(*)) AS n FROM d GROUP BY %s ORDER BY %s", rootQry, field, field, field)
	rdr := s.NewReader(qry, conn)
	if e := rdr.Init("", chutils.MergeTree); e != nil {
		return nil, e
	}

	hd := &HistData{}
	rows, _, e := rdr.Read(0, false)
	if e != nil {
		return nil, e
	}

	for ind := 0; ind < len(rows); ind++ {
		hd.Levels = append(hd.Levels, rows[ind][0])
		hd.Counts = append(hd.Counts, rows[ind][1].(int32))
	}

	return hd, nil
}

func (hd *HistData) String() string {
	return strings.Join(Aligner(hd.Levels, hd.Counts, 5), "\n")
}

func (hd *HistData) Histogram() *grob.Fig {
	histPlot := &grob.Bar{X: hd.Levels, Y: hd.Counts, Type: grob.TraceTypeBar}
	fig := &grob.Fig{Data: grob.Traces{histPlot}}

	return fig
}
