package cams

import (
	"math"
	"strings"
	"testing"
)

func TestBasic(t *testing.T) {
	type testfixture struct {
		name string
		result string
		f func(g *camvars, n Coord)
		params Coord
	}

    tests := []testfixture {
		{"moveto", "G0 X1.2000 Y3.4000", (*camvars).Moveto , Coord{1.2, 3.4, math.NaN()} },
		{"moveby", "G91\nG0 X1.2000 Y3.4000", (*camvars).Moveby , Coord{1.2, 3.4, math.NaN()} },
		{"moveby2", "G0 X1.2000 Y3.4000", (*camvars).Moveby , Coord{1.2, 3.4, math.NaN()} },
		{"moveto2", "G90\nG0 X1.2000 Y3.4000", (*camvars).Moveto , Coord{1.2, 3.4, math.NaN()} },
		{"movetoX", "G0 X1.2000", (*camvars).Moveto , X(1.2) },
		{"movetoY", "G0 Y1.2000", (*camvars).Moveto , Y(1.2) },
		{"movetoZ", "G0 Z1.2000", (*camvars).Moveto , Z(1.2) },
		{"movetoXY", "G0 X1.2000 Y3.4000", (*camvars).Moveto , XY(1.2, 3.4) },
		{"movetoXYZ", "G0 X1.2000 Y3.4000 Z5.6000", (*camvars).Moveto , XYZ(1.2, 3.4, 5.6) },
		{"movetoYZ", "G0 Y3.4000 Z5.6000", (*camvars).Moveto , YZ(3.4, 5.6) },
	}

	var b strings.Builder
    g := NewCam(&b)
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
      		tt.f(g, tt.params)
		    if strings.TrimRight(b.String(), "\n ") != tt.result { t.Errorf("Expected <%s> but got <%s>", tt.result, b.String()) }
        })
        b.Reset()
    }
}

func TestRectangle(t *testing.T) {
	var b strings.Builder
    g := NewCam(&b)

    g.Laser = false
    g.Tool_diameter = 4
    g.Doc = 1
    var args = map[string]float64 {"width": 12.0, "length": 3.4, "depth": 2, "feedrate": 1000.0}
    err := g.Rectangle(false, args)
    if err != nil {
       	t.Errorf("got Error %v, want %v", err, nil)
       	t.FailNow()
    }

result :=
`(Rectangle: 12.0000x3.4000 mm, inside: false, depth: 2.0000 mm, doc: 1.0000, tool: 4.0000)
G91
G0 X-2.0000Y-2.0000
G1 F1800.0
G90
G1 Z0.0000
G91
G1 X16.0000Z-1.0000
G1 Y7.4000
G1 X-16.0000
G1 Y-7.4000
G1 X16.0000Z-1.0000
G1 Y7.4000
G1 X-16.0000
G1 Y-7.4000
G1 X16.0000
G90
G0 Z20.0000
G91
G0 X-16.0000
G90
(-- Rectangle --)
`
	if b.String() != result { t.Errorf("Did not get Expected result:\n<%v>\ngot:\n<%v>", result, b.String()) }

}
