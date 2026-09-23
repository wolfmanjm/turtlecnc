package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	cams "github.com/wolfmanjm/turtlecnc/internal"
)

func main() {
    // get flags
    debug := flag.Bool("t", false, "debug test")
    server := flag.Bool("s", false, "start server")
    port := flag.Int("port", 8080, "Set server listen port")

    flag.Parse()

    if *debug {
	   cams.Testit()
    } else if *server {
        runServer(*port)
    } else {
        flag.PrintDefaults()
    }
}

// a structure containing all possible values from the request
type values struct {
    Kind string `json:"kind"`
    Jobname string `json:"jobname"`
    Length float64 `json:"length"`
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
    Stepover float64
    Laser bool
    Conventional bool
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
        return
    }

    var v values
    err := json.NewDecoder(r.Body).Decode(&v)
    if err != nil {
        http.Error(w, "JSON decode Error", http.StatusInternalServerError)
        log.Println(fmt.Sprint("JSON Decode error: %w", err))
        return
    }

    // fmt.Printf("%+v\n", v)

    kind := v.Kind

    w.Header().Set("Content-Type", "application/text")

    // handle each job kind
    switch kind {
    case "rectangle":
        log.Println(fmt.Sprintf("We got a rectangle job, values: %+v", v))
        rectangleJob(w, v)
    case "hole":
        log.Println(fmt.Sprintf("We got a hole job: %+v", v))
        holeJob(w, v)
    case "sidemill":
        log.Println(fmt.Sprintf("We got a sidemill job: %+v", v))
        sidemillJob(w, v)
    case "squaring":
        log.Println(fmt.Sprintf("We got a squaring job: %+v", v))
        squaringJob(w, v)
    case "circle":
        log.Println(fmt.Sprintf("We got a circle job: %+v", v))
    default:
        log.Println(fmt.Sprintf("Unknown job: %v, %+v", kind, v))
        fmt.Fprint(w, fmt.Sprintf("Unknown job: %v", kind))
        return
    }
}

func rectangleJob(w io.Writer, v values) {
    // depth:6 doc:1.5 feedrate:1400 jobname:rectangle length:150 rpm:10000 side:laser tooldia:6.35 width:100
    g := cams.NewCam(w)

    g.Laser = false
    g.Preamble()

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

    g.Tool_diameter = v.Tooldia
    g.Doc = v.Doc
    g.RPM = v.RPM

    var args = map[string]float64 {
        "width": v.Width,
        "length": v.Length,
        "depth": v.Depth,
        "feedrate": v.Feedrate,
    }

     if err := g.Rectangle(inside, args); err != nil {
        fmt.Fprintf(w, "Rectangle() failed: %s", err.Error())
    }
}

func holeJob(w io.Writer, v values) {
    // depth:12 feedrate:1400 holedia:5 jobname:hole laser:false peck:0 rpm:10000 tooldia:6.35
    g := cams.NewCam(w)

    g.Laser = v.Laser
    g.Preamble()

    if !g.Laser {
        g.Tool_diameter = v.Tooldia
        g.Doc = v.Doc
        g.RPM = v.RPM
    }

    g.Feedrate = v.Feedrate

    var args = map[string]float64 {
        "diameter": v.Diameter,
        "depth": v.Depth,
        "peck": v.Peck,
    }

     if err := g.Hole(args); err != nil {
        fmt.Fprintf(w, "Hole() failed: %v", err.Error())
    }
}

func sidemillJob(w io.Writer, v values) {
    // {Kind:sidemill Jobname:surface Length:300 Width:0 Diameter:0 Depth:1 RPM:0 Side: Tooldia:12 Doc:0.5 Peck:0 Feedrate:1400 Laser:false}
    g := cams.NewCam(w)
    g.Preamble()

    g.Tool_diameter = v.Tooldia
    g.Doc = v.Doc
    g.RPM = v.RPM
    g.Feedrate = v.Feedrate

    if err := g.Sidemill(v.Side, v.Conventional, v.Length, v.Depth); err != nil {
        fmt.Fprintf(w, "Sidemill() failed: %s", err.Error())
    }
}

func squaringJob(w http.ResponseWriter, v values) {
    g := cams.NewCam(w)
    g.Preamble()

    g.Tool_diameter = v.Tooldia
    g.Doc = v.Doc
    g.RPM = v.RPM
    g.Feedrate = v.Feedrate

    if err := g.Squaring(v.Width, v.Length, v.Height, v.Depth, v.Stepover); err != nil {
        fmt.Fprintf(w, "Squaring() failed: %s", err.Error())
    }
}

func runServer(port int) {
    fileserver := http.FileServer(http.Dir("./static"))

    http.HandleFunc("/api/dialog-result", apiHandler)
    http.Handle("/", fileserver)

    p := fmt.Sprintf(":%d", port)
    log.Printf("port running on http://localhost:%d/\n", port)
    if err := http.ListenAndServe(p, nil); err != nil{
        log.Fatal(err)
    }
}

