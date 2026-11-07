package main

import (
	"fmt"
	cams "github.com/wolfmanjm/turtlecnc/internal"
	"io"
	"net/http"
)

// a structure containing all possible values from the request
type values struct {
    Kind string
    Jobname string
    Length float64
    Width float64
    Diameter float64
    Depth float64
    Height float64
    Doc float64
    RPM float64
    Side string
    Tooldia float64
    Peck float64
    Feedrate float64
    PlungeRate float64
    Stepover float64
    Laser bool
    Conventional bool
    Spiral bool
}

func rectangleJob(w io.Writer, v values) {
	// depth:6 doc:1.5 feedrate:1400 jobname:rectangle length:150 rpm:10000 side:laser tooldia:6.35 width:100
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
		fmt.Fprintf(w, "Unknown side: %v", v.Side)
		return
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

	if err := g.Rectangle(inside, args); err != nil {
		fmt.Fprintf(w, "Rectangle() failed: %s", err.Error())
	}
}

func circleJob(w io.Writer, v values) {
	// depth:6 doc:1.5 feedrate:1400 jobname:rectangle length:150 rpm:10000 side:laser tooldia:6.35 width:100
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
		fmt.Fprintf(w, "Unknown side: %v", v.Side)
		return
	}

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	var args = map[string]float64{
		"diameter": v.Diameter,
		"depth":    v.Depth,
		"segments": v.Stepover,
	}

	if err := g.Circle(inside, args); err != nil {
		fmt.Fprintf(w, "Circle() failed: %s", err.Error())
	}
}

func holeJob(w io.Writer, v values) {
	// depth:12 feedrate:1400 holedia:5 jobname:hole laser:false peck:0 rpm:10000 tooldia:6.35
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

	if err := g.Hole(v.Diameter, v.Depth, v.Peck); err != nil {
		fmt.Fprintf(w, "Hole() failed: %v", err.Error())
	}
}

func sidemillJob(w io.Writer, v values) {
	// {Kind:sidemill Jobname:surface Length:300 Width:0 Diameter:0 Depth:1 RPM:0 Side: Tooldia:12 Doc:0.5 Peck:0 Feedrate:1400 Laser:false}
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	if err := g.Sidemill(v.Side, v.Conventional, v.Length, v.Depth); err != nil {
		fmt.Fprintf(w, "Sidemill() failed: %s", err.Error())
	}
}

func surfaceJob(w io.Writer, v values) {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate

	g.Preamble()

	if err := g.Milltop(v.Width, v.Length, v.Depth, v.Stepover); err != nil {
		fmt.Fprintf(w, "Surface() failed: %s", err.Error())
	}
}

func squaringJob(w http.ResponseWriter, v values) {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate
	g.PlungeRate = v.PlungeRate

	g.Preamble()

	if err := g.Squaring(v.Width, v.Length, v.Height, v.Depth, v.Stepover); err != nil {
		fmt.Fprintf(w, "Squaring() failed: %s", err.Error())
	}
}

func pocketJob(w http.ResponseWriter, v values) {
	g := cams.NewCam(w)

	g.ToolDiameter = v.Tooldia
	g.Doc = v.Doc
	g.RPM = v.RPM
	g.Feedrate = v.Feedrate
	g.PlungeRate = v.PlungeRate
	g.Preamble()

	if err := g.Pocket(v.Spiral, v.Width, v.Length, v.Depth, v.Stepover); err != nil {
		fmt.Fprintf(w, "Pocket() failed: %s", err.Error())
	}
}
