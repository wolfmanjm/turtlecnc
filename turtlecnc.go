package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

    cams "github.com/wolfmanjm/turtlecnc/internal"
)

func main() {
	//cams.Testit()
    runServer()
}

// a structure containing all possible values from the request
type values struct {
    Kind string `json:"kind"`
    Jobname string `json:"jobname"`
    Length float64 `json:"length"`
    Width float64
    Diameter float64
    Depth float64
    RPM float64
    Side string
    Tooldia float64
    Doc float64
    Peck float64
    Feedrate float64
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
    case "circle":
        log.Println(fmt.Sprintf("We got a circle job: %+v", v))
    default:
        log.Println(fmt.Sprintf("Unknown job: %v, %+v", kind, v))
        fmt.Fprintf(w, fmt.Sprintf("Unknown job: %v", kind))
        return
    }
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
        fmt.Fprintf(w, "Rectangle() failed: %w", err.Error())
    }
}

func holeJob(w io.Writer, v values) {
    // depth:12 feedrate:1400 holedia:5 jobname:hole laser:false peck:0 rpm:10000 tooldia:6.35
    g := cams.NewCam(w)

    g.Laser = v.Laser

    if !g.Laser {
        g.Tool_diameter = v.Tooldia
        g.Doc = v.Doc
        g.RPM = v.RPM
    }

    var args = map[string]float64 {
        "diameter": v.Diameter,
        "depth": v.Depth,
        "feedrate": v.Feedrate,
        "peck": v.Peck,
    }

     if err := g.Hole(args); err != nil {
        fmt.Fprintf(w, "Hole() failed: %v", err.Error())
    }
}

func sidemillJob(w io.Writer, v values) {
// {Kind:sidemill Jobname:surface Length:300 Width:0 Diameter:0 Depth:1 RPM:0 Side: Tooldia:12 Doc:0 Peck:0 Feedrate:1400 Laser:false}
    g := cams.NewCam(w)

    g.Tool_diameter = v.Tooldia
    g.Doc = v.Doc
    g.RPM = v.RPM

    var args = map[string]float64 {
        "length": v.Length,
        "depth": v.Depth,
        "feedrate": v.Feedrate,
    }

     if err := g.Sidemill(v.Side, v.Conventional, args); err != nil {
        fmt.Fprintf(w, "Sidemill() failed: %w", err.Error())
    }
}

func runServer() {
    fileserver := http.FileServer(http.Dir("./static"))

    http.HandleFunc("/api/dialog-result", apiHandler)
    http.Handle("/", fileserver)

    log.Printf("port running on http://localhost:8080/\n")
    if err := http.ListenAndServe(":8080", nil); err != nil{
        log.Fatal(err)
    }
}

