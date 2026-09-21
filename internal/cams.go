package cams

import (
	"fmt"
	"io"
	"math"
	"os"
)

type camvars struct {
	Safe_z        float64
	Tool_diameter float64
	Doc           float64
	Feedrate      float64
	Plunge_rate   float64
	default_depth float64
	Laser         bool
	Lathe         bool
	abs_state	  bool
	output_port   io.Writer
}

func NewCam(out io.Writer) *camvars {
    var g camvars

    g.Laser = false
    g.Lathe = false
    g.Feedrate = 1800
    g.Plunge_rate = 0
    g.Safe_z = 20
    g.Tool_diameter = 3.175
    g.Doc = 0.2
    g.default_depth = 6.35
	g.abs_state = true
	g.output_port = out

	return &g
}


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

func (g camvars) output(s ...string) {
    for _, o := range s {
        fmt.Fprintf(g.output_port, o)
    }
}

func (g camvars) outputnl(s ...string) {
    g.output(s...)
    g.output("\n")
}

func (g *camvars) Absolute()() {
	if !g.abs_state {
		g.outputnl("G90")
		g.abs_state = true
	}
}

func (g *camvars) Relative() {
	if g.abs_state {
		g.outputnl("G91")
		g.abs_state = false
	}
}

func (g camvars) Pause(msg string) {
	g.outputnl("(MSG ", msg, ")")
	g.outputnl("M0")
}

func (g camvars) move(gc string, coords Coord) {

	g.output(gc, " ")
    if !math.IsNaN(coords.X)  {
		g.output(fmt.Sprintf("X%.4f ", coords.X))
	}
    if !math.IsNaN(coords.Y)  {
        g.output(fmt.Sprintf("Y%.4f ", coords.Y))
    }
    if !math.IsNaN(coords.Z)  {
        g.output(fmt.Sprintf("Z%.4f ", coords.Z))
    }
	g.outputnl()
}

func (g *camvars) Moveto(l Coord) {
	g.Absolute()
	g.move("G0", l)
}

func (g *camvars) Moveby(l Coord) {
	g.Relative()
	g.move("G0", l)
}

func (g *camvars) Cutto(l Coord) {
	g.Absolute()
	g.move("G1", l)
}

func (g *camvars) Cutby(l Coord) {
	g.Relative()
	g.move("G1", l)
}

func (g camvars) SetFeedrate(fr float64) {
    g.outputnl(fmt.Sprintf("G1 F%.1f", fr))
}

// shrink a rectangle by d
func _shrink(w, l, d float64) (float64, float64, float64, float64) {
    return d/2, d/2, w-d, l-d
}

func _expand(w, l, d float64) (float64, float64, float64, float64) {
    return -d/2, -d/2, w+d, l+d
}

// profiles a rectangle of given size, inside or outside to the given depth using the depth of cut
func (g *camvars) Rectangle(inside bool, args map[string]float64) error {
    // cuts a rectangle conventional milling, ramps to depth in the first X line, but recuts to final depth as last operation
    var w, l, fr float64
    var ok bool
    var doc = g.Doc

    if w, ok = args["width"]; !ok {
        return fmt.Errorf("width required")
    }

    if l, ok = args["length"]; !ok {
        return fmt.Errorf("length required")
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = g.Feedrate
    }

    if g.Laser {
        g.outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, for Laser)", w, l))
        if !math.IsNaN(fr) {
            g.SetFeedrate(fr)
        }
        g.Cutby(X(w))
        g.Cutby(Y(l))
        g.Cutby(X(-w))
        g.Cutby(Y(-l))

    } else {
        depth := -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        g.outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", w, l, inside, -depth, doc, g.Tool_diameter))

        // shrink rectangle if inside cut or expand if outside cut

        var xoff, yoff float64
        if inside {
            xoff, yoff, w, l = _shrink(w, l, g.Tool_diameter)
        } else {
            xoff, yoff, w, l = _expand(w, l, g.Tool_diameter)
        }

        // adjust start point for tool diameter
        g.Moveby(XY(xoff, yoff))
        // set feedrate
        g.SetFeedrate(fr)
        // Move to surface
        g.Cutto(Z(0))

        var z float64
        if doc >= math.Abs(depth) {
            doc = math.Abs(depth)
            z = depth
        } else {
            z = -doc
        }

        done := false
        for !done {
            g.Cutby(XZ(w, -doc))
            g.Cutby(Y(l))
            g.Cutby(X(-w))
            g.Cutby(Y(-l))

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
                g.Cutby(X(w))
                g.Moveto(Z(g.Safe_z))
                g.Moveby(X(-w))
            }
        }
    }
    g.Absolute()
    g.outputnl("(-- Rectangle --)")
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
func (g *camvars) Circle(inside bool, args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var segments int
    var doc = g.Doc

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
        fr = g.Feedrate
    }

    if g.Laser {
        g.outputnl(fmt.Sprintf("(Circle: radius %.4f mm, segments %v for laser)", radius, segments))
        pa := _circle(radius, segments)
        g.SetFeedrate(fr)
        // We are starting at the center of the circle so need to move to the edge to start
        // the points returned by _circle are absolute coordinates around 0,0 so we calculate the delta for each segment
        p0 := pa[0]
        g.Moveby(XY(p0.x, p0.y))
        var last point
        for i, pp := range pa {
            if i > 0 {
                p1 := last.sub(pp)
                g.Cutby(XY(p1.x, p1.y))
            }
            last = pp
        }
        g.Moveby(XY(-p0.x, -p0.y))

    } else {
        depth := -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }

        g.outputnl(fmt.Sprintf("(Circle: radius %.4f mm, segments %v, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", radius, segments, inside, -depth, doc, g.Tool_diameter))

        if inside {
            radius = radius - g.Tool_diameter/2.0
        } else {
            radius = radius + g.Tool_diameter/2.0
        }

        pa := _circle(radius, segments)
        g.SetFeedrate(fr)

        // We are starting at the center of the circle so need to move to the edge to start
        // the points returned by _circle are absolute coordinates around 0,0 so we calculate the delta for each segment
        p0 := pa[0]
        g.Moveby(XY(p0.x, p0.y))

        var z float64
        if doc >= math.Abs(depth) {
            doc = math.Abs(depth)
            z = depth
        } else {
            z = -doc
        }

        // Move to surface
        g.Cutto(Z(0))

        done := false
        for !done {
            g.Cutby(Z(-doc))
            var last point
            for i, pp := range pa {
                if i > 0 {
                    p1 := last.sub(pp)
                    g.Cutby(XY(p1.x, p1.y))
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

        g.Moveto(Z(g.Safe_z))
        g.Moveby(XY(-p0.x, -p0.y))
    }

    g.Absolute()
    g.outputnl("(-- Circle --)")
    return nil
}

func (g *camvars) Hole(args map[string]float64) error {
    var radius, fr float64
    var ok bool
    var doc = g.Doc

    if radius, ok = args["radius"]; !ok {
        if radius, ok = args["diameter"]; !ok {
            return fmt.Errorf("radius or diameter required")
        }
        radius = radius / 2
    }

    if !g.Laser && radius*2.0 <= g.Tool_diameter {
        return fmt.Errorf("cannot make holes smaller than tool diameter %v", g.Tool_diameter)
    }

    if fr, ok = args["feedrate"]; !ok {
        fr = g.Feedrate
    }

    var off, depth float64
    if g.Laser {
        off = radius
        g.outputnl(fmt.Sprintf("(Hole diameter: %.4f mm for laser)", radius*2))

    } else {
        off = (radius - g.Tool_diameter/2.0) // offset due to tool diameter
        depth = -g.default_depth
        if i, ok := args["depth"]; ok {
            depth = -i
        }
        g.outputnl(fmt.Sprintf("(Hole diameter: %.4f mm, depth: %.4f, doc: %.4f)", radius*2, -depth, doc))
    }

    var peck float64
    if peck, ok = args["peck"]; !ok {
        peck = 0
        if radius*2.0 <= g.Tool_diameter+2 {
            fmt.Println("(WARNING: you may need to add the peck parameter to make holes so close to the tool diameter)")
        }
    } else {
        g.outputnl(fmt.Sprintf("(Peck set to: %.4f mm)", peck))
    }

    // move to start point
    g.Moveby(X(off))
    g.Absolute()

    if g.Laser {
        g.outputnl(fmt.Sprintf("G2 J0 I%.4f F%.1f", -off, fr))
        g.Moveby(X(-off))

    } else {
        g.Moveto(Z(0))
        z := 0.0
        g.SetFeedrate(fr)

        if math.Abs(depth) < doc {
            z = depth
        } else {
            z = -doc
        }

        delta := 0.0
        done := false

        // spiral
        for !done {
            g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, z))
            delta += doc
            // see if we need to peck to get rid of swarf
            if peck > 0 && delta > peck {
                delta = 0
                g.Moveto(Z(g.Safe_z))
                g.outputnl("G4 S1")
                g.Cutto(Z(z))
            }

            z -= doc

            if z < depth {
                // we do a complete circle to make sure the bottom is flat
                g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, depth))
                g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -off, depth))
                done = true
            }
        }
    }

    g.Moveto(Z(g.Safe_z))
    // center again
    g.Moveby(X(-off))
    g.Absolute()

    g.outputnl("(-- Hole --)")
    return nil
}

func Testit() {
    g := NewCam(os.Stdout)

    g.Moveto(Coord{1.2, 3.4, math.NaN()})
    g.Moveby(Coord{2, 4, math.NaN()})
    g.Moveby(Coord{3, 5, math.NaN()})
    g.Moveto(Coord{1.2, 3.4, math.NaN()})
    g.Pause("test it")

    g.Laser = false
    g.Tool_diameter = 4
    g.Doc = 1
    var args = map[string]float64 {"width": 12.0, "length": 3.4, "depth": 2, "feedrate": 1000.0}
    if err:= g.Rectangle(false, args); err != nil {
        panic(err)
    }

    g.Moveto(XYZ(20, 20, 0))
    g.Laser = false
    args = map[string]float64 {"radius": 10.0, "segments": 64, "depth": 2, "feedrate": 1000.0}
    if err:= g.Circle(false, args); err != nil {
        panic(err)
    }

    //Moveto(XYZ(0, 0, 0))
    g.Moveto(Z(0))
    g.Laser = false
    g.Doc = 0.2
    args = map[string]float64 {"radius": 5.0, "depth": 2, "peck": 1, "feedrate": 1000.0}
    if err:= g.Hole(args); err != nil {
        panic(err)
    }
}
