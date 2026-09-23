package cams

import (
	"encoding/json"
	"log"
	"math"
	"os"
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

// merge the camvars structure with the json supplied
func merge(s1 *camvars, s2 string) {
	err := json.NewDecoder(strings.NewReader(s2)).Decode(s1)
	if err != nil {
	    panic(err)
	}
}

type testfixture1 struct {
	name string
	resultfile string
	inside bool
	args map[string]float64
	settings string
}

func validate(t *testing.T, resultfile string, result string) {
	data, err := os.ReadFile("./testdata/" + resultfile)
	if err != nil {
		log.Fatal(err)
	}

	// Split the file content into lines
	test_lines := strings.Split(string(data), "\n")
	// Split the result content into lines
	result_lines := strings.Split(result, "\n")

	// Test each line, trim white space at end of line
	for i, line := range test_lines {
		if strings.TrimRight(line, "\n ") != strings.TrimRight(result_lines[i], "\n ") {
			t.Errorf("Did not get Expected result at line %d:\nexpected:\n<%v>\ngot:\n<%v>", i+1, line, result_lines[i])
			t.FailNow()
		}
	}
}

func TestRectangle(t *testing.T) {
    tests := []testfixture1 {
    	{	"rectangle_mill", "rectangle1.txt", false,
    		map[string]float64{"width": 12.0, "length": 3.4, "depth": 2, "feedrate": 1000.0},
    		`{"laser":false,"tool_diameter":4.0,"doc":1.0}` },

    	{	"rectangle_laser", "rectangle2.txt", false,
    		map[string]float64{"width": 100.0, "length": 150.0, "feedrate": 1234.0},
    		`{"laser":true}` },

	   	{	"rectangle_inside", "rectangle3.txt", true,
	    		map[string]float64{"width": 100.0, "length": 100.0, "depth": 1.0},
	    		`{"tool_diameter":6.35,"doc":0.2}` },
	}

	// test function
	f := func(t *testing.T, tt testfixture1) {
		// captures result
		var b strings.Builder
		// create context
    	g := NewCam(&b)
    	// merge in overides
    	merge(g, tt.settings)
    	// call cam
	    err := g.Rectangle(tt.inside, tt.args)
	    if err != nil {
	       	t.Errorf("got Error %v, want %v", err, nil)
	       	t.FailNow()
	    }
	    // validate results
	    validate(t, tt.resultfile, b.String())
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
        	f(t, tt)
        })
    }

}
