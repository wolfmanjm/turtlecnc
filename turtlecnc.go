package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
)

// globals used by many of the jobs
// TODO run cam jobs in thread and use thread local storage?
type globals struct {
	safe_z        float64
	tool_diameter float64
	doc           float64
	feedrate      float64
	plunge_rate   float64
	default_depth float64
	laser         bool
	lathe         bool
}

func (g *globals) newGlobals() {
    g.laser = false
    g.lathe = false
    g.feedrate = 1800
    g.plunge_rate = 0
    g.safe_z = 20
    g.tool_diameter = 3.175
    g.doc = 0.2
    g.default_depth = 6.35
}

// this cannot be a global it needs to be request local
var abs_state = true

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
	//testit()
    runServer()
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
        return
    }

    buf, err := io.ReadAll(r.Body)
    if err != nil {
        fmt.Println()
        http.Error(w, fmt.Sprint("readall error: %w", err), http.StatusInternalServerError)
        return
    }

    m := make(map[string]any)
    json.Unmarshal(buf, &m)

    kind, ok := m["kind"]
    if !ok {
        http.Error(w, fmt.Sprint("reply does not have a kind"), http.StatusInternalServerError)
        return
    }

    values, ok := m["values"].(map[string]any)
     if !ok {
        http.Error(w, fmt.Sprint("reply does not have any values"), http.StatusInternalServerError)
        return
    }

    // handle each job kind
    // TODO make this a goroutine
    switch kind {
    case "rectangle":
        fmt.Println(fmt.Sprintf("We got a rectangle job, values: %v", values))
        w.Header().Set("Content-Type", "application/text")
        rectangleJob(w, values)
    case "hole":
        fmt.Println("We got a hole job")
    case "circle":
        fmt.Println("We got a circle job")
    default:
        fmt.Println("Unknown job: ", kind)
        return
    }
}

func rectangleJob(w io.Writer, v map[string]any) {
    // depth:6 doc:1.5 feedrate:1400 jobname:rectangle length:150 rpm:10000 side:laser tooldia:6.35 width:100
    var g globals
    g.newGlobals()

    g.laser = false

    inside := true
    switch v["side"] {
    case "inside":
        inside = true
    case "outside":
        inside = false
    case "laser":
        g.laser = true
    default:
        fmt.Fprintf(w, "Unknown side: %v", v["side"])
        return
    }

    g.tool_diameter = v["tooldia"].(float64)
    g.doc = v["doc"].(float64)

    var args = map[string]float64 {
        "width": v["width"].(float64),
        "length": v["length"].(float64),
        "depth": v["depth"].(float64),
        "feedrate": v["feedrate"].(float64),
    }

    var err error
    // FIXME this needs to be context
    output_port = w
    defer resetOutput()

    if err = Rectangle(g, inside, args); err != nil {
        fmt.Fprintf(w, "Rectangle() failed: %w", err)
        return
    }
}

func resetOutput() {
	output_port = os.Stdout
}

func runServer() {
    fileserver := http.FileServer(http.Dir("./static"))

    http.HandleFunc("/api/dialog-result", apiHandler)
    http.Handle("/", fileserver)

    fmt.Printf("port running on http://localhost:8080/\n")
    if err := http.ListenAndServe(":8080", nil); err != nil{
        log.Fatal(err)
    }
}

var output_port io.Writer = os.Stdout

func output(s ...string) {
    for _, o := range s {
        fmt.Fprintf(output_port, o)
    }
}

func outputnl(s ...string) {
    output(s...)
    output("\n")
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
func Rectangle(g globals, inside bool, args map[string]float64) error {
    // cuts a rectangle conventional milling, ramps to depth in the first X line, but recuts to final depth as last operation
    var w, l, fr float64
    var ok bool
    var doc = g.doc

    if w, ok = args["width"]; !ok {
        return fmt.Errorf("width required")
    }

    if l, ok = args["length"]; !ok {
        return fmt.Errorf("length required")
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = g.feedrate
    }

    if g.laser {
        outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, for Laser)", w, l))
        if !math.IsNaN(fr) {
            SetFeedrate(fr)
        }
        Cutby(X(w))
        Cutby(Y(l))
        Cutby(X(-w))
        Cutby(Y(-l))

    } else {
        depth := -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", w, l, inside, -depth, g.doc, g.tool_diameter))

        // shrink rectangle if inside cut or expand if outside cut

        var xoff, yoff float64
        if inside {
            xoff, yoff, w, l = _shrink(w, l, g.tool_diameter)
        } else {
            xoff, yoff, w, l = _expand(w, l, g.tool_diameter)
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
                Moveto(Z(g.safe_z))
                Moveby(X(-w))
            }
        }
    }
    Absolute()
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
func Circle(g globals, inside bool, args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var segments int
    var doc = g.doc

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
        fr = g.feedrate
    }

    if g.laser {
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
        depth := -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        outputnl(fmt.Sprintf("(Circle: radius %.4f mm, segments %v, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", radius, segments, inside, -depth, doc, g.tool_diameter))

        if inside {
            radius = radius - g.tool_diameter/2.0
        } else {
            radius = radius + g.tool_diameter/2.0
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

        Moveto(Z(g.safe_z))
        Moveby(XY(-p0.x, -p0.y))
    }

    Absolute()
    return nil
}

func Hole(g globals, args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var doc = g.doc

    if radius, ok = args["radius"]; !ok {
        if radius, ok = args["diameter"]; !ok {
            return fmt.Errorf("radius or diameter required")
        }
        radius = radius / 2
    }

    if !g.laser && radius*2.0 <= g.tool_diameter {
        return fmt.Errorf("cannot make holes smaller than tool diameter %v", g.tool_diameter)
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = g.feedrate
    }

    var off, depth float64
    if g.laser {
        off = radius
        outputnl(fmt.Sprintf("(Hole diameter: %.4f mm for laser)", radius*2))

    } else {
        off = (radius - g.tool_diameter/2.0) // offset due to tool diameter
        depth = -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }
        outputnl(fmt.Sprintf("(Hole diameter: %.4f mm, depth: %.4f, doc: %.4f)", radius*2, -depth, doc))
    }

    var peck float64
    if peck, ok = args["peck"]; !ok {
        peck = 0
        if radius*2.0 <= g.tool_diameter+2 {
            fmt.Println("(WARNING: you may need to add the peck parameter to make holes so close to the tool diameter)")
        }
    } else {
        outputnl(fmt.Sprintf("(Peck set to: %.4f mm)", peck))
    }

    // move to start point
    Moveby(X(off))
    Absolute()

    if g.laser {
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
                Moveto(Z(g.safe_z))
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

    Moveto(Z(g.safe_z))
    // center again
    Moveby(X(-off))
    Absolute()

    return nil
}

func testit() {
    Moveto(Coord{1.2, 3.4, math.NaN()})
    Moveby(Coord{2, 4, math.NaN()})
    Moveby(Coord{3, 5, math.NaN()})
    Moveto(Coord{1.2, 3.4, math.NaN()})
    Pause("test it")

    var g globals
    g.newGlobals()

    g.laser = false
    g.tool_diameter = 4
    g.doc = 1
    var args = map[string]float64 {"width": 12.0, "length": 3.4, "depth": 2, "feedrate": 1000.0}
    if err:= Rectangle(g, false, args); err != nil {
        panic(err)
    }

    Moveto(XYZ(20, 20, 0))
    g.laser = false
    args = map[string]float64 {"radius": 10.0, "segments": 64, "depth": 2, "feedrate": 1000.0}
    if err:= Circle(g, false, args); err != nil {
        panic(err)
    }

    //Moveto(XYZ(0, 0, 0))
    Moveto(Z(0))
    g.laser = false
    g.doc = 0.2
    args = map[string]float64 {"radius": 5.0, "depth": 2, "peck": 1, "feedrate": 1000.0}
    if err:= Hole(g, args); err != nil {
        panic(err)
    }
}
