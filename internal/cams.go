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
	RPM 		  float64
	Laser         bool
	Lathe         bool
	abs_state	  bool
	output_port   io.Writer
}

func NewCam(out io.Writer) *camvars {
    var g camvars

    // set defaults
    g.Laser = false
    g.Lathe = false
    g.Feedrate = 1800
    g.Plunge_rate = 0
    g.Safe_z = 20
    g.Tool_diameter = 3.175
    g.Doc = 0.2
    g.RPM = 0
	g.abs_state = true
	g.output_port = out

	return &g
}

// Helpers for setting coordinates
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

// output to specified port
func (g camvars) output(s ...string) {
    for _, o := range s {
        fmt.Fprint(g.output_port, o)
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
    var w, l float64
    var ok bool
    var doc = g.Doc

    w, ok = args["width"]
    if !ok || w == 0 {
        return fmt.Errorf("width required")
    }

    l, ok = args["length"]
    if !ok || l == 0 {
        return fmt.Errorf("length required")
    }

    fr := g.Feedrate

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
		if doc <= 0 {
    		return fmt.Errorf("Depth of cut must be > 0")
    	}

    	var depth float64
        if depth, ok = args["depth"]; !ok {
            return fmt.Errorf("depth required")
        }
        depth = -depth

        g.outputnl(fmt.Sprintf("(Rectangle: %.4fx%.4f mm, inside: %v, depth: %.4f mm, doc: %.4f, tool: %.4f)", w, l, inside, -depth, doc, g.Tool_diameter))

        // shrink rectangle if inside cut or expand if outside cut

        var xoff, yoff float64
        if inside {
            xoff, yoff, w, l = _shrink(w, l, g.Tool_diameter)
        } else {
            xoff, yoff, w, l = _expand(w, l, g.Tool_diameter)
        }

    	if g.RPM > 0.001 {
     		g.outputnl(fmt.Sprintf("M3 S%.1f", g.RPM))
     		g.outputnl("G4 S3")
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
               	if g.RPM > 0.001 {
		     		g.outputnl("M5")
		     	}
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
    var radius float64
    var ok bool
    var segments int
    var doc = g.Doc

    if radius, ok = args["radius"]; !ok || radius == 0 {
        if radius, ok = args["diameter"]; !ok || radius == 0 {
            return fmt.Errorf("radius or diameter required")
        }
        radius = radius * 2
    }

    if s, ok := args["segments"]; ok && s > 0 {
        segments = int(s)
    } else {
        segments = int(math.Round(radius*2 * math.Pi / 0.5))
    }

    fr := g.Feedrate

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
		if doc <= 0 {
    		return fmt.Errorf("Depth of cut must be > 0")
    	}

    	var depth float64
        if depth, ok = args["depth"]; !ok {
            return fmt.Errorf("depth required")
        }
        depth = -depth

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

func (g *camvars) Hole(diameter, depth, peck float64) error {
    var radius float64
    var doc = g.Doc

    if diameter == 0 {
        return fmt.Errorf("diameter required")
    }
    radius = diameter / 2

    fr := g.Feedrate

    var off, o float64
    need_multiple_passes := false

    if g.Laser {
        off = radius
        g.outputnl(fmt.Sprintf("(Hole diameter: %.4f mm for laser)", radius*2))

    } else {
        off = (radius - g.Tool_diameter/2.0) // offset due to tool diameter
	    if off <= 0 {
	        return fmt.Errorf("cannot make holes smaller than tool diameter %v", g.Tool_diameter)
	    }

	    if doc <= 0 {
	    	return fmt.Errorf("Depth of cut must be > 0")
	    }

        if depth == 0 {
        	return fmt.Errorf("Depth required")
        }
        depth = -depth

        g.outputnl(fmt.Sprintf("(Hole diameter: %.4f mm, depth: %.4f, doc: %.4f)", radius*2, -depth, doc))

	    if peck == 0 {
	        if radius*2.0 <= g.Tool_diameter+1 {
	            fmt.Println("(WARNING: you may need to add the peck parameter to make holes so close to the tool diameter)")
	        }
	    } else {
	        g.outputnl(fmt.Sprintf("(Peck set to: %.4f mm)", peck))
	    }

       	if g.RPM > 0.001 {
     		g.outputnl(fmt.Sprintf("M3 S%.1f", g.RPM))
     		g.outputnl("G4 S3")
     	}

	    // if the hole is larger than twice diameter of tool we need to increase
	    // radius of cut until we reach required diameter so we do not leave
	    // a plug that can be forcefully ejected
	    if diameter > g.Tool_diameter*2 {
	    	o = g.Tool_diameter/2
	    	need_multiple_passes = true
	    } else {
    		o = off
    	}
    }


    if g.Laser {
	    // move to start point
	    g.Moveby(X(off))
	    g.Absolute()
        g.SetFeedrate(fr)
        g.outputnl(fmt.Sprintf("G2 J0 I%.4f", -off))

     } else {
	    // move to start point
	    g.Moveby(X(o))
        g.Moveto(Z(0))
        z := 0.0
        g.SetFeedrate(fr)

        if math.Abs(depth) < doc {
            z = depth
        } else {
            z = -doc
        }

        dz := z
        delta := 0.0
        done := false

        // spiral down to depth
        for !done {
            g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -o, dz))
            delta += doc
            // see if we need to peck to get rid of swarf
            if peck > 0 && delta > peck {
                delta = 0
                g.Moveto(Z(g.Safe_z))
                g.outputnl("G4 S1")
                g.Cutto(Z(dz))
            }

            dz -= doc

            if dz < depth {
                // we do a complete circle to make sure the bottom is flat
                g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -o, depth))
                g.outputnl(fmt.Sprintf("G2 J0 I%.4f Z%.4f", -o, depth))

                if need_multiple_passes {
	                delta = 0
    	            g.Moveby(X(-o)) // center it
    	            g.Moveto(Z(1)) // move up

    	            dz = z
    	            o += g.Tool_diameter/2
    	            if o >= off {
    	            	o = off
    	            	need_multiple_passes = false
    	            }
				    // move to new start point
	    			g.Moveby(X(o))
    	            g.Moveto(Z(0))

                } else {
                	done = true
                }
            }
        }

    	g.Moveto(Z(g.Safe_z))

       	if g.RPM > 0.001 {
     		g.outputnl("M5")
     	}
    }

    // center again
    g.Moveby(X(-off))
    g.Absolute()

    g.outputnl("(-- Hole --)")
    return nil
}

 func (g *camvars) Milltop(w, l, depth, stepover float64) error {
 	var overlap float64
    if stepover > 0 {
    	overlap = (100.0-stepover)/100.0
    } else {
    	overlap = 0.4
    }

    fr := g.Feedrate

    g.outputnl(fmt.Sprintf("(Milltop %.4fx%.4f mm, depth: %.4f mm, tool: %.4f, stepover: %.2f)", w, l, depth, g.Tool_diameter, overlap))
	g.outputnl("(MSG Position tool at front left of surface just touching it, set Z0")

	if g.RPM > 0.001 {
 		g.outputnl(fmt.Sprintf("M3 S%.1f", g.RPM))
 		g.outputnl("G4 S3")
 	}

    // do perimeter first
    g.SetFeedrate(fr)
    g.Cutby(Z(-depth))
    g.Cutby(X(w))
    g.Cutby(Y(l))
    g.Cutby(X(-w))
    g.Cutby(Y(-l))

    // mill inside grid
    inc := g.Tool_diameter * overlap
    y := inc
    dir := 1.0
    for y <= l {
    	g.Cutby(Y(inc))
    	g.Cutby(X(w*dir))
    	y += inc
    	dir = -dir  // change direction
    }

    if y != l {
      d := (y-l)
      g.Cutby(Y(d))
      g.Cutby(X(w*dir))
    }

    g.Moveto(Z(g.Safe_z))
   	if g.RPM > 0.001 {
   		g.outputnl("M5")
   	}

    g.outputnl("(-- Milltop --)")
 	return nil
 }

/*
	Side mill a specific edge
	bottom edge, or top edge, or left edge, or right edge
	can set climb or conventional milling, and uses relative moves from starting position
	must also set tool diameter and doc
*/
func (g *camvars) Sidemill(edge string, conventional bool, l, depth float64) error {
    if l == 0 || depth == 0 {
        return fmt.Errorf("length and depth required")
    }

    fr := g.Feedrate

    var (
    	axis string
    	delta float64
    	moveaxis string
    	movedelta float64
    	start string
    )

    switch edge {
    case "bottom":
    	axis = "Y"
    	delta = +1
    	moveaxis = "X"
    	if conventional {
    		movedelta = +1;
    		start = "Left"
    	} else {
    		movedelta = -1
	    	start = "Right"
	    }
    case "top":
    	axis = "Y"
    	delta = -1
    	moveaxis = "X"
    	if conventional {
    		movedelta = -1;
    		start = "Right"
    	} else {
    		movedelta = +1
	    	start = "left"
	    }

    case "left":
    	axis = "X"
    	delta = +1
    	moveaxis = "Y"
    	if conventional {
    		movedelta = -1;
    		start = "Top"
    	} else {
    		movedelta = +1
	    	start = "Bottom"
	    }

    case "right":
    	axis = "X"
    	delta = -1
    	moveaxis = "Y"
     	if conventional {
    		movedelta = +1;
    		start = "Bottom"
    	} else {
    		movedelta = -1
	    	start = "Top"
	    }

    default:
    	return fmt.Errorf("edge must be one of top, bottom, left, right")
    }

    t := "Conventional"
    if !conventional {
    	t = "Climb"
    }

	doc := g.Doc
    if doc <= 0 {
    	return fmt.Errorf("Depth of cut must be > 0")
    }

    g.outputnl(fmt.Sprintf("(Sidemill %.4f mm along %v edge to depth of %.4f mm, %v)", l, edge, depth, t))
	g.outputnl(fmt.Sprintf("(MSG Move cutter to touch the %v edge and move to %v corner, and move to the required height)", edge, start))

	if g.RPM > 0.001 {
 		g.outputnl(fmt.Sprintf("M3 S%.1f", g.RPM))
 		g.outputnl("G4 S3")
 	}

	retract := 1.0
	extra := g.Tool_diameter
	d := depth

	g.SetFeedrate(fr)
	g.Relative()

	for d > 0 {
		d -= g.Doc
		if d < 0 {
			doc = (d + g.Doc)
			d = 0
		}
		retract += doc

		// move depth of cut
		g.outputnl(fmt.Sprintf("G1 %v%.4f", axis, doc*delta))
		// skim edge
		g.outputnl(fmt.Sprintf("G1 %v%.4f", moveaxis, (l+extra) * movedelta))
		// move off edge
		g.outputnl(fmt.Sprintf("G0 %v%.4f", axis,  retract * -delta))
		// move back to start point
		g.outputnl(fmt.Sprintf("G0 %v%.4f", moveaxis, (l+extra) * -movedelta))
		// move back to edge
		g.outputnl(fmt.Sprintf("G0 %v%.4f", axis,retract * delta))
	}

   	if g.RPM > 0.001 {
 		g.outputnl("M5")
 	}

	g.Absolute()
	g.outputnl("(-- Sidemill --)")
	return nil
}

/*
  This is used for squaring stock on the mill. as per Joe Pie on Youtube...
  It uses a clockwise path (climb milling) and goes around the stock and mills in the usual pattern (around clockwise).
  The origin should be set at front left of actual stock, and width height is set to actual size of stock
  however the milling bit will move to the left and right of the stock to move between top and bottom raster
  if final_depth is set it will continue to take doc off until it reaches that depth.
  If height is set then it will side mill around the perimeter at the height given using width of cut the same as doc
*/
func (g *camvars) Squaring(w, l, height, depth, stepover float64) error {

   	if w == 0 || l == 0  {
        return fmt.Errorf("width, length required")
    }

    var overlap float64
    if stepover > 0 {
    	overlap = (100.0-stepover)/100.0
    } else {
    	overlap = 0.4
    }

    fr := g.Feedrate
    doc := g.Doc
    z := -doc

	g.outputnl(fmt.Sprintf("(Squaring: %.4fx%.4f mm, doc: %.4f mm, to final depth: %.4f)", w, l, doc, depth))
	if height > 0 {
		g.outputnl(fmt.Sprintf("(Sidemill at depth: %.4f mm with doc: %.4f mm)", height, doc))
	}
	g.outputnl("(MSG touch off at the bottom left corner and set XY to zero)")

    // set the Y increment on each pass based on tool size and overlap set
    inc := g.Tool_diameter * overlap
    // set the start x and end x such that the tool clears the stock before it moves up and down
    xs := -g.Tool_diameter // only needs to be half but make sure we totally are clear of stock
    xe := w + g.Tool_diameter
    y1 := l + g.Tool_diameter/2.0 - inc  // initial engage at top is overlap
    y2 := -g.Tool_diameter/2.0 + inc       // initial engage at bottom is overlap

    g.Moveto(Z(g.Safe_z))

 	if g.RPM > 0.001 {
 		g.outputnl(fmt.Sprintf("M3 S%.1f", g.RPM))
 		g.outputnl("G4 S3")
 	}

    // move to required starting position
    g.Moveto(XY(xs, y1))
    // move to first cutting depth
    g.SetFeedrate(fr) // maybe should be plunge feed rate
    g.Cutto(Z(-doc))

    g.outputnl("(first top skim pass)")
    for {
    	for {
	        // mill top raster
        	g.Cutto(X(xe))
	        // move to bottom raster
        	g.Moveto(Y(y2))
        	// mill bottom raster
        	g.Cutto(X(xs))

	        // calculate next raster
	        y1 -= inc
	        y2 += inc

	        if y1 <= y2 { break }
            // move to next top raster
        	g.Moveto(Y(y1))
    	}

    	// if we have no final depth we are done
	    if depth == 0 { break }
      	if z <= -depth { break }

		// start again at new depth until we reach final
		y1 = l + g.Tool_diameter/2.0 - inc  // initial engage at top is overlap
		y2 = -g.Tool_diameter/2.0 + inc     // initial engage at bottom is overlap

	    // move to required starting position
	    g.Moveto(Y(y1))

        // move to new cutting depth
    	z -= doc
      	if z < -depth {
        	z = -depth
      	}
    	g.outputnl(fmt.Sprintf("(skim pass at %.4f)", z))
      	g.Moveto(Z(z))
    }

   	// See if we need to sidemill the edges as per Joe Pie
    // it tries to take up backlash presuming home has been set in +X +Y direction
    if height > 0 {
    	// should woc be a separate parameter?
    	smwoc := (-g.Tool_diameter/2.0) + doc
    	g.outputnl("(skim edges)")
    	g.Moveto(Z(g.Safe_z))
	    // move to a position to take up backlash
		g.Moveto(XY(-g.Tool_diameter, -g.Tool_diameter))
	    // move to required cutting depth
	    g.Cutto(Z(-height))
   	  	g.Cutto(X(smwoc))

   	  	// skim left edge
   	  	g.Cutto(Y(l-smwoc))
    	// skim top edge to left edge
   	  	g.Cutto(X(w-smwoc))
    	// skim right edge all the way down past end
   	  	g.Cutto(Y(-g.Tool_diameter))
    	// skim bottom edge move back to it to take up backlash
   	  	g.Cutto(Y(smwoc))

   	  	g.Cutto(X(-g.Tool_diameter))
   	  	// leave to the left and bottom
	    g.Moveto(XY(-g.Tool_diameter, -g.Tool_diameter))
    }

    g.Moveto(Z(g.Safe_z))
   	if g.RPM > 0.001 {
 		g.outputnl("M5")
 	}
	g.Absolute()

	g.outputnl("(-- Squaring --)")
	return nil
}

func (g *camvars) Pocket(width, length, depth, stepover float64) error {

   	if width == 0 || length == 0 || depth == 0 {
        return fmt.Errorf("width, length, depth required")
    }

    var overlap float64
    if stepover > 0 {
    	overlap = (100.0-stepover)/100.0
    	if overlap == 0 { overlap = 0.1 }
    } else {
    	overlap = 0.4
    }

    doc := g.Doc

	g.outputnl(fmt.Sprintf("(Pocket: %.4fx%.4fx%.4f mm, doc: %.4f mm)", width, length, depth, doc))
	g.outputnl("(MSG Touch off bottom left corner of pocket and surface)")

    z := 0.0
    delta := g.Tool_diameter * overlap

    g.SetFeedrate(g.Feedrate)
    g.Moveto(XY(g.Tool_diameter/2, g.Tool_diameter/2))
    g.Moveto(Z(0))
    var w, l float64
    // start at corner of the pocket and work inwards
    for z > -depth {
 		var xoff, yoff float64
		_, _, w, l = _shrink(width, length, g.Tool_diameter)
     	z -= doc
    	if z < -depth {
    		doc = -depth - z
    		z = -depth
    	}

    	// ramps into work
    	zd := -doc/4
    	g.Cutby(YZ(l, zd)) // left side
    	g.Cutby(XZ(w, zd)) // top side
    	g.Cutby(YZ(-l, zd)) // right side
    	g.Cutby(XZ(-w, zd)) // bottom side

     	for {
    		xoff, yoff, w, l = _shrink(w, l, delta*2)
    		if w <= 0 || l <= 0 { break }
	    	// now move to next circuit
	    	g.Cutby(XY(xoff, yoff)) // in by overlap amount
	    	g.Cutby(Y(l)) // left side
	    	g.Cutby(X(w)) // top side
	    	g.Cutby(Y(-l)) // right side
	    	g.Cutby(X(-w)) // bottom side
		}
	    g.Moveto(XY(g.Tool_diameter/2, g.Tool_diameter/2))
    }

    // do last pass to get to final depth
	_, _, w, l = _shrink(width, length, g.Tool_diameter)
    g.Moveto(Z(-depth))
	g.Cutby(Y(l)) // left side
	g.Cutby(X(w)) // top side
	g.Cutby(Y(-l)) // right side
	g.Cutby(X(-w)) // bottom side


    g.Moveto(Z(g.Safe_z))
    g.Moveto(XY(0, 0))
    g.Absolute()
    g.outputnl("(-- pocket --)")
    return nil
}

func (g *camvars) Preamble() {
	g.outputnl("(Begin preamble)")
	g.outputnl("G17 G90")
	g.outputnl("G21")
	g.abs_state = true
	if ! g.Laser {
		g.outputnl(fmt.Sprintf("(Tool diameter: %.4f mm, Feedrate: %.2f mm/min)", g.Tool_diameter, g.Feedrate))
	}
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

    g.Laser = true
    args = map[string]float64 {"width": 100.0, "length": 150, "feedrate": 1234.0}
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
    g.Tool_diameter = 4
    if err:= g.Hole(8.0, 2, 1); err != nil {
        panic(err)
    }

    g.Moveto(Z(0))
    g.Laser = false
    g.Doc = 0.2
    if err:= g.Hole(13.0, 2, 0); err != nil {
        panic(err)
    }

    g.Moveto(XYZ(0, 0, 0))
    g.Tool_diameter = 50
    if err:= g.Milltop(100, 200, 1, 50); err != nil {
        panic(err)
    }

    g.Moveto(XYZ(0, 0, 0))
    g.Tool_diameter = 12
    g.Doc = 0.2
    if err:= g.Sidemill("bottom", false, 200, 0.5); err != nil {
        panic(err)
    }

    g.Moveto(XYZ(0, 0, 0))
    g.Tool_diameter = 12
    g.Doc = 0.2
    if err:= g.Squaring(100, 100, 10, 1, 40); err != nil {
        panic(err)
    }

    g.Moveto(XYZ(0, 0, 0))
    g.Tool_diameter = 6.35 / 2.0
    g.Preamble()
    g.Doc = 0.2
    if err:= g.Pocket(30, 25, 1, 40); err != nil {
        panic(err)
    }

}
