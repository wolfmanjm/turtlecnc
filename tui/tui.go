package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	cams "github.com/wolfmanjm/turtlecnc/internal"
)

// Fixed content sizes (inner size, excluding the border).
const (
	listWidth = 30
	formWidth = 44
	// 1 gap + 1 button row,
	// plus the form's 1-row top and bottom padding.
	formHeight = 1 + 1 + 2
)

// center wraps a primitive in flex spacers so it sits in the middle of the
// screen at exactly width x height (outer size, including its border).
func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}

var app *tview.Application
var pages *tview.Pages
var Debug bool

func RunTui() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	// escape at any time will always exit
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.Stop()
		}
		return event
	})

	items := []string {
		"Hole", "Spiral cut hole",
		"Surface", "Flatten a surface",
		"Squaring", "Square a block",
		"Sidemill", "Sidemill one edge",
		"Rectangle", "Profile a rectangle",
		"Circle", "Profile a circle",
		"Pocket", "rectangular pocket",
	}

	// ---- Job List ----
	list := tview.NewList().ShowSecondaryText(true)
	list.SetBorder(true).SetTitle(" Select a Job ")
	for i := 0; i < len(items); i+=2 {
		list.AddItem(items[i], items[i+1], rune('1'+i/2), nil)
	}
	list.AddItem("Quit", "Press to exit", 'q', func() {
		app.Stop()
	})

	// when the job is selected build the form for it
	list.SetSelectedFunc(func(_ int, name, _ string, _ rune) {
		if name != "Quit" {
			form, nitems := makeForm(name)
			pages.AddAndSwitchToPage("form", center(form, formWidth+2, nitems*2+formHeight+1), true)
		}
	})

	// Border adds 2 to each dimension.
	pages.AddPage("list", center(list, listWidth+2, len(items)+4), true, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func makeForm(title string) (*tview.Form, int) {
	form := tview.NewForm()
	form.SetBorder(true)

	form.Clear(true)

	job, ok := Jobs[title]
	if !ok {
		panic(fmt.Sprintf("Job type: %s Not found", title))
	}
	form.SetTitle(job.title)

	//form.AddTextView("", job.blurb, 0, 0, false, false)

	fields := job.fields

	maxWidth := 0
	for i := range fields {
		k := fields[i].label
		v := fields[i].value
		if strings.Contains(v, "|") {
			options := strings.Split(v, "|")
			form.AddDropDown(k, options, 0, nil)
		} else if v == "true" || v == "false" {
			var c bool
			if v == "true" { c = true }
			form.AddCheckbox(k, c, nil)
		} else {
			form.AddInputField(k, v, 0, nil, nil)
			if len(k) > maxWidth { maxWidth = len(k) }
		}
	}

	//fmt.Fprintln(os.Stderr, maxWidth)

	form.AddButton("OK", func() {
		err := handleResults(title, form)
		if err != nil {
			showMessage(fmt.Sprintf("Error: %s", err.Error()), "form")
		} else {
			if Debug {
				// for debug to see the output
				app.Suspend(func() {
				    fmt.Println(GCodeOutput.String())
				    fmt.Print("Hit Enter to continue")
	    			var input string
	    			fmt.Scanln(&input)
				})
				showMessage("Job Done OK", "list")
			} else {
				fn := form.GetFormItemByLabel("Job name").(*tview.InputField).GetText()
				showMessage(fmt.Sprintf("Job OK saved to: %s.nc", fn), "list")
			}
		}
	} )
	form.AddButton("Help", func() { showHelp(title) } )
	form.AddButton("Quit", func() { app.Stop() } )
	form.SetCancelFunc(func() { app.Stop() } )
	form.SetFocus(0)
	// add extra navigation keys
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '?' {
			showHelp(title)
		}
	/*
		if event.Key() == tcell.KeyDown {
			f, _ := form.GetFocusedItemIndex()
			if f >= 0 {
				form.SetFocus(f+1)
			}
		} else if event.Key() == tcell.KeyUp {
			f, _ := form.GetFocusedItemIndex()
			if f > 0 {
				form.SetFocus(f-1)
			}
		}
	*/
		return event
	})
	return form, len(fields)
}

func showHelp(title string) {
	var helptext strings.Builder
	j, ok := Jobs[title]
	if !ok || j.blurb == "" {
		helptext.WriteString("No help found")
	} else {
		helptext.WriteString(j.blurb)
	}
	helptext.WriteString("\n\nEntries:\n")
	var nlines int
	for _, v := range j.fields {
		helptext.WriteString(fmt.Sprintf("  [blue]%s[white] %s\n", v.label, v.unit))
		nlines++
	}
	helptext.WriteString("\n[yellow]Press Enter to Continue")

	tv := tview.NewTextView()
	tv.SetBorder(true)
	//tv.SetLabel("Help For job " + title)
	tv.SetDynamicColors(true).
		SetWordWrap(true).
		SetText(helptext.String()).
		SetDoneFunc(func(key tcell.Key) {
			pages.RemovePage("help")
			pages.SwitchToPage("form")
		})

	width := 70
	nlines += len(j.blurb) / width
	pages.AddAndSwitchToPage("help", center(tv, width, nlines + 8), true)
}

// a structure containing all possible values from the form
type values struct {
    Kind string
    Jobname string `json:"job name"`
    Length float64
    Width float64
    Diameter float64
    Depth float64
    Height float64
    Doc float64 `json:"depth of cut"`
    RPM float64 `json:"spindle"`
    Side string
    Tooldia float64 `json:"tool diameter"`
    Peck float64 `json:"peck depth"`
    Feedrate float64
    PlungeRate float64
    Stepover float64
    Segments float64
    Laser bool
    Conventional bool
    Spiral bool
}

func handleResults(title string, form *tview.Form) error {
	val := func(label string) string {
		i := form.GetFormItemByLabel(label)
		switch i.(type) {
		case *tview.Checkbox:
			if i.(*tview.Checkbox).IsChecked() {
				return "true"
			} else {
				return "false"
			}
		case *tview.DropDown:
			_, v := i.(*tview.DropDown).GetCurrentOption()
			return v
		default:
			return i.(*tview.InputField).GetText()
		}
	}
	// convert results to a Json map string
	var results strings.Builder
	results.WriteString("{ ")
	for i := 0; i < form.GetFormItemCount(); i++ {
		fi := form.GetFormItem(i)
		v := val(fi.GetLabel())
		// v is a string and we need to output it as an unquoted string if it is a number
		// and as true/false if a bool and quoted if it is anything else
		var format string
		if v == "" {
			v = "0"
			format = "%q: %v"
		} else if isNumber(v) || v == "true" || v == "false" {
			format = "%q: %v"
		} else {
			format = "%q: %q"
		}
		results.WriteString(fmt.Sprintf(format, fi.GetLabel(), v ))
		if i < form.GetFormItemCount() - 1 {
			results.WriteString(",\n")
		}
	}
	results.WriteString(" }\n")

	// json decode into the values structure
    var v values
	err := json.NewDecoder(strings.NewReader(results.String())).Decode(&v)
    if err != nil {
        panic(fmt.Sprintf("JSON Decode error: %v -  %s", err, results.String()))
    }

    return handleJob(title, v)
}

// check if the string is a number
func isNumber(s string) bool {
    dotFound := false
    for _, v := range s {
        if v == '.' {
            if dotFound {
                return false
            }
            dotFound = true
        } else if v < '0' || v > '9' {
            return false
        }
    }
    return true
}

// popup message box
func showMessage(results, page string) {
	modal := tview.NewModal()
	modal.SetText(results).
		AddButtons([]string{"Continue", "Quit"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				app.Stop()
			} else {
				pages.RemovePage("info")
				if page == "" {
					app.Stop()
				} else {
					pages.SwitchToPage(page)
				}
			}
		})
	pages.AddAndSwitchToPage("info", modal, true)
}

// only used for Debug output
var GCodeOutput strings.Builder
func handleJob(kind string, v values) error {
	var w io.Writer
	if Debug {
		GCodeOutput.Reset()
		w = &GCodeOutput

	} else {
		ofn := v.Jobname + ".nc"
	   	fw, err := os.Create(ofn)
	   	if err != nil {
	   		panic(err)
	   	}
	   	defer fw.Close()
	   	w = fw
	}

	var err error = nil
    // handle each job kind
    switch kind {
    case "Rectangle":
        err = rectangleJob(w, v)
    case "Hole":
        err = holeJob(w, v)
    case "Sidemill":
        err = sidemillJob(w, v)
    case "Surface":
        err = surfaceJob(w, v)
    case "Squaring":
        err = squaringJob(w, v)
    case "Pocket":
        err = pocketJob(w, v)
    case "Circle":
        err = circleJob(w, v)
    default:
        panic(fmt.Sprintf("Unknown job: %v", kind))
    }

    return err
}

// hand off to the cams package
func surfaceJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	return g.Milltop(v.Width, v.Length, v.Depth, v.Stepover)
}

func holeJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	if !v.Laser {
		g.ToolDiameter = v.Tooldia
		g.Doc = v.Doc
		g.RPM = v.RPM
	}

	g.Laser = v.Laser
	g.Feedrate = v.Feedrate

	g.Preamble()

	g.Feedrate = v.Feedrate

	return g.Hole(v.Diameter, v.Depth, v.Peck)
}

func rectangleJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.Laser = false

	inside := true
	switch v.Side {
	case "inside":
		inside = true
	case "outside":
		inside = false
	case "laser":
		g.Laser = true
	default:
		return fmt.Errorf("Unknown side: %v", v.Side)
	}

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	var args = map[string]float64{
		"width":  v.Width,
		"length": v.Length,
		"depth":  v.Depth,
	}

	return  g.Rectangle(inside, args)
}

func circleJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.Laser = false

	inside := true
	switch v.Side {
	case "inside":
		inside = true
	case "outside":
		inside = false
	case "laser":
		g.Laser = true
	default:
		return fmt.Errorf("Unknown side: %v", v.Side)
	}

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	var args = map[string]float64{
		"diameter": v.Diameter,
		"depth":    v.Depth,
		"segments": v.Segments,
	}

	return g.Circle(inside, args)
}

func sidemillJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	return g.Sidemill(v.Side, v.Conventional, v.Length, v.Depth)
}

func squaringJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate
	g.PlungeRate = v.PlungeRate

	g.Preamble()

	return g.Squaring(v.Width, v.Length, v.Height, v.Depth, v.Stepover)
}

func pocketJob(w io.Writer, v values) error {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate
	g.PlungeRate = v.PlungeRate
	g.Preamble()

	return g.Pocket(v.Spiral, v.Width, v.Length, v.Depth, v.Stepover)
}

