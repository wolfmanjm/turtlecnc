package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	_ "strings"
)

// globals used by many of the jobs
var (
	safe_z        float64
	tool_diameter float64
	doc           float64
	feedrate      float64
	plunge_rate   float64
	default_depth float64
	laser         bool
	lathe         bool
)

// local state
var abs_state     bool

type Coord struct {
    X, Y, Z float64
}
// Helpers for setting specific coordinates
func X(x float64) Coord {
    return Coord{x, math.NaN(), math.NaN()}
}
func Y(y float64) Coord {
    return Coord{math.NaN(), y, math.NaN()}
}
func Z(z float64) Coord {
    return Coord{math.NaN(), math.NaN(), z}
}
func XY(x, y float64) Coord {
    return Coord{x, y, math.NaN()}
}
func XZ(x, z float64) Coord {
    return Coord{x, math.NaN(), z}
}
func XYZ(x, y, z float64) Coord {
    return Coord{x, y, z}
}
func YZ(y, z float64) Coord {
    return Coord{math.NaN(), y, z}
}


func main() {
	laser = false
	lathe = false
	feedrate = 1800
	plunge_rate = 0
	safe_z = 20
	tool_diameter = 3.175
	doc = 0.2
	default_depth = 6.35
	abs_state = true

	testit()
    //runServer()
}

func runServer() {
    fileserver := http.FileServer(http.Dir("./static"))

    http.Handle("/", fileserver)

    fmt.Printf("port running on http://localhost:8080/\n")
    if err := http.ListenAndServe(":8080", nil); err != nil{
        log.Fatal(err)
    }
}

// output functions, TODO redirect to a file or to web write
func output(s ...string) {
	for _, o := range s {
		fmt.Print(o)
	}
}

func outputnl(s ...string) {
	output(s...)
	fmt.Println()
}

func g90() {
	if !abs_state {
		outputnl("G90")
		abs_state = true
	}
}

func g91() {
	if abs_state {
		outputnl("G91")
		abs_state = false
	}
}

func Pause(msg string) {
	outputnl("(MSG ", msg, ")")
	outputnl("M0")
}

func Absolute() {
	g90()
}

func Relative() {
	g91()
}

func move(g string, coords Coord) {

	output(g, " ")
    if !math.IsNaN(coords.X)  {
		output(fmt.Sprintf("X%.4f ", coords.X))
	}
    if !math.IsNaN(coords.Y)  {
        output(fmt.Sprintf("Y%.4f ", coords.Y))
    }
    if !math.IsNaN(coords.Z)  {
        output(fmt.Sprintf("Z%.4f ", coords.Z))
    }
	outputnl()
}

func Moveto(l Coord) {
	g90()
	move("G0", l)
}

func Moveby(l Coord) {
	g91()
	move("G0", l)
}

func Cutto(l Coord) {
	g90()
	move("G1", l)
}

func Cutby(l Coord) {
	g91()
	move("G1", l)
}

func SetFeedrate(fr float64) {
    outputnl(fmt.Sprintf("G1 F%.1f", fr))
}

// shrink a rectangle by d
func _shrink(w, l, d float64) (float64, float64, float64, float64) {
    return d/2, d/2, w-d, l-d
}

func _expand(w, l, d float64) (float64, float64, float64, float64) {
    return -d/2, -d/2, w+d, l+d
}

// profiles a rectangle of given size, inside or outside to the given depth using the depth of cut
func Rectangle(inside bool, args map[string]float64) error {
    // cuts a rectangle conventional milling, ramps to depth in the first X line, but recuts to final depth as last operation
    var w, l, fr float64
    var ok bool
    var save_doc = doc

    if w, ok = args["width"]; !ok {
        return fmt.Errorf("width required")
    }

    if l, ok = args["length"]; !ok {
        return fmt.Errorf("length required")
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = feedrate
    }

    if laser {
        outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, for Laser)", w, l))
        if !math.IsNaN(fr) {
            SetFeedrate(fr)
        }
        Cutby(X(w))
        Cutby(Y(l))
        Cutby(X(-w))
        Cutby(Y(-l))

    } else {
        depth := -default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", w, l, inside, -depth, doc, tool_diameter))

        // shrink rectangle if inside cut or expand if outside cut

        var xoff, yoff float64
        if inside {
            xoff, yoff, w, l = _shrink(w, l, tool_diameter)
        } else {
            xoff, yoff, w, l = _expand(w, l, tool_diameter)
        }

        // adjust start point for tool diameter
        Moveby(XY(xoff, yoff))
        // set feedrate
        SetFeedrate(fr)
        // Move to surface
        Cutto(Z(0))

        var z float64
        if doc >= math.Abs(depth) {
            doc = math.Abs(depth)
            z = depth
        } else {
            z = -doc
        }

        done := false
        for !done {
            Cutby(XZ(w, -doc))
            Cutby(Y(l))
            Cutby(X(-w))
            Cutby(Y(-l))

            if z <= depth {
              done = true
            } else if (z-doc) >= depth {
                z -= doc
            } else {
              doc = math.Abs(z - depth)
              z = depth
            }

            if done {
                // we do first line again to get it completely to depth
                Cutby(X(w))
                Moveto(Z(safe_z))
                Moveby(X(-w))
            }
        }
    }
    Absolute()
    doc = save_doc
    return nil
}

// generates the coordinates for a circle based around 0,0 with a given radius
type point struct { x, y float64 }
func (p point) sub(p2 point) point {
    var t point
    t.x = p2.x - p.x
    t.y = p2.y - p.y
    return t
}

func _circle(radius float64, segments int) []point {
    points := make([]point, 0, segments)
    var p point
    for a := 0; a <= segments; a++ {
        p.x = math.Sin(float64(a) * (360.0 / float64(segments)) * math.Pi / 180.0) * radius
        p.y = math.Cos(float64(a) * (360.0 / float64(segments)) * math.Pi / 180.0) * radius
        points = append(points, p)
    }
    return points
}

// Profiles a circle of given radius, inside or outside
func Circle(inside bool, args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var segments int
    var save_doc = doc

    if radius, ok = args["radius"]; !ok {
        if radius, ok = args["diameter"]; !ok {
            return fmt.Errorf("radius or diameter required")
        }
        radius = radius * 2
    }

    if s, ok := args["segments"]; ok {
        segments = int(s)
    } else {
        segments = int(math.Round(radius*2 * math.Pi / 0.5))
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = feedrate
    }

    if laser {
        outputnl(fmt.Sprintf("(Circle: radius %.4f mm, segments %v for laser)", radius, segments))
        pa := _circle(radius, segments)
        SetFeedrate(fr)
        // We are starting at the center of the circle so need to move to the edge to start
        // the points returned by _circle are absolute coordinates around 0,0 so we calculate the delta for each segment
        p0 := pa[0]
        Moveby(XY(p0.x, p0.y))
        var last point
        for i, pp := range pa {
            if i > 0 {
                p1 := last.sub(pp)
                Cutby(XY(p1.x, p1.y))
            }
            last = pp
        }
        Moveby(XY(-p0.x, -p0.y))

    } else {
        depth := -default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        outputnl(fmt.Sprintf("(Circle: radius %.4f mm, segments %v, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", radius, segments, inside, -depth, doc, tool_diameter))

        if inside {
            radius = radius - tool_diameter/2.0
        } else {
            radius = radius + tool_diameter/2.0
        }

        pa := _circle(radius, segments)
        SetFeedrate(fr)

        // We are starting at the center of the circle so need to move to the edge to start
        // the points returned by _circle are absolute coordinates around 0,0 so we calculate the delta for each segment
        p0 := pa[0]
        Moveby(XY(p0.x, p0.y))

        var z float64
        if doc >= math.Abs(depth) {
            doc = math.Abs(depth)
            z = depth
        } else {
            z = -doc
        }

        // Move to surface
        Cutto(Z(0))

        done := false
        for !done {
            Cutby(Z(-doc))
            var last point
            for i, pp := range pa {
                if i > 0 {
                    p1 := last.sub(pp)
                    Cutby(XY(p1.x, p1.y))
                }
                last = pp
            }

            if z <= depth {
              done = true
            } else if (z-doc) >= depth {
                z -= doc
            } else {
              doc = math.Abs(z - depth)
              z = depth
            }
        }

        Moveto(Z(safe_z))
        Moveby(XY(-p0.x, -p0.y))
    }

    Absolute()
    doc = save_doc
    return nil
}

func Hole(args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var save_doc = doc

    if radius, ok = args["radius"]; !ok {
        if radius, ok = args["diameter"]; !ok {
            return fmt.Errorf("radius or diameter required")
        }
        radius = radius / 2
    }

    if !laser && radius*2.0 <= tool_diameter {
        return fmt.Errorf("cannot make holes smaller than tool diameter %v", tool_diameter)
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = feedrate
    }

    var off, depth float64
    if laser {
        off = radius
        outputnl(fmt.Sprintf("(Hole diameter: %.4f mm for laser)", radius*2))

    } else {
        off = (radius - tool_diameter/2.0) // offset due to tool diameter
        depth = -default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }
        outputnl(fmt.Sprintf("(Hole diameter: %.4f mm, depth: %.4f, doc: %.4f)", radius*2, -depth, doc))
    }

    var peck float64
    if peck, ok = args["peck"]; !ok {
        peck = 0
        if radius*2.0 <= tool_diameter+2 {
            fmt.Println("(WARNING: you may need to add the peck parameter to make holes so close to the tool diameter)")
        }
    } else {
        outputnl(fmt.Sprintf("(Peck set to: %.4f mm)", peck))
    }

    // move to start point
    Moveby(X(off))
    Absolute()

    if laser {
        outputnl(fmt.Sprintf("G2 J0 I%.4f F%.1f", -off, fr))
        Moveby(X(-off))

    } else {
        Moveto(Z(0))
        z := 0.0
        SetFeedrate(fr)

        if math.Abs(depth) < doc {
            z = depth
        } else {
            z = -doc
        }

        delta := 0.0
        done := false

        // spiral
        for !done {
            outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, z))
            delta += doc
            // see if we need to peck to get rid of swarf
            if peck > 0 && delta > peck {
                delta = 0
                Moveto(Z(safe_z))
                outputnl("G4 S1")
                Cutto(Z(z))
            }

            z -= doc

            if z < depth {
                // we do a complete circle to make sure the bottom is flat
                outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, depth))
                outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, depth))
                done = true
            }
        }
    }

    Moveto(Z(safe_z))
    // center again
    Moveby(X(-off))
    Absolute()
    doc = save_doc
    return nil
}

func testit() {
    Moveto(Coord{1.2, 3.4, math.NaN()})
    Moveby(Coord{2, 4, math.NaN()})
    Moveby(Coord{3, 5, math.NaN()})
    Moveto(Coord{1.2, 3.4, math.NaN()})
    Pause("test it")

    laser = false
    tool_diameter = 4
    doc = 1
    var args = map[string]float64 {"width": 12.0, "length": 3.4, "depth": 2, "feedrate": 1000.0}
    if err:= Rectangle(false, args); err != nil {
        panic(err)
    }

    Moveto(XYZ(20, 20, 0))
    laser = false
    args = map[string]float64 {"radius": 10.0, "segments": 64, "depth": 2, "feedrate": 1000.0}
    if err:= Circle(false, args); err != nil {
        panic(err)
    }

    //Moveto(XYZ(0, 0, 0))
    Moveto(Z(0))
    laser = false
    doc = 0.2
    args = map[string]float64 {"radius": 5.0, "depth": 2, "peck": 1, "feedrate": 1000.0}
    if err:= Hole(args); err != nil {
        panic(err)
    }
}
