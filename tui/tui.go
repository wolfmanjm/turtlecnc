package tui

import (
	"strings"
	"fmt"
	"os"

	"github.com/rivo/tview"
)

// Fixed content sizes (inner size, excluding the border).
const (
	listWidth = 30
	formWidth = 44
	// 6 fields + 5 gaps between them + 1 gap + 1 button row,
	// plus the form's 1-row top and bottom padding.
	formHeight = 2 + 5 + 1 + 1 + 2
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

func RunTui() {
	app = tview.NewApplication()
	pages = tview.NewPages()

	items := []string {
		"Hole", "Spiral cut hole",
		"Milltop", "Flatten a surface",
		"Squaring", "Square a block",
		"Sidemill", "Sidemill one edge",
		"Rectangle", "Profile a rectangle",
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
		form, nitems := makeForm(name)
		pages.AddAndSwitchToPage("form", center(form, formWidth+2, nitems+formHeight+2), true)
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
	form.SetTitle(title)

	// form entries with optional default values
	var entries []string
	if title == "Milltop" {
		entries = []string {
			"job name", "Mill top",
			"width", "",
			"length", "",
			"tool diameter", "6.35",
			"depth", "",
			"feedrate", "1200",
			"overlap", "40",
			"plunge rate", "100", }
	} else {
		entries = []string {
			"job name", "test",
			"width", "",
			"length", "", }
	}

	maxWidth := 0
	for i := 0; i < len(entries); i+=2 {
		k := entries[i]
		form.AddInputField(k, entries[i+1], 0, nil, nil)
		if len(k) > maxWidth { maxWidth = len(k) }
	}

	//fmt.Fprintln(os.Stderr, maxWidth)

	form.AddButton("OK", func() { handleResults(form) } )
	form.AddButton("Quit", func() { app.Stop() } )
	form.SetCancelFunc(func() { app.Stop() } )
	form.SetFocus(0)

	return form, len(entries)/2
}

func handleResults(form *tview.Form) {
	val := func(label string) string {
		return form.GetFormItemByLabel(label).(*tview.InputField).GetText()
	}
	var results strings.Builder
	for i := 0; i < form.GetFormItemCount(); i++ {
		fi := form.GetFormItem(i)
		results.WriteString(fmt.Sprintf("%s: %s\n", fi.GetLabel(), val(fi.GetLabel()) ))
	}

	showMessage(results.String())
}

func showMessage(results string) {
	modal := tview.NewModal()
	modal.SetText(results).
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				app.Stop()
			} else {
				pages.SwitchToPage("form")
			}
		})
	pages.AddAndSwitchToPage("info", modal, true)
}
