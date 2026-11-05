package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
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
    PlungeRate float64
    Stepover float64
    Laser bool
    Conventional bool
    Spiral bool
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    // Log the request URI and method
    log.Printf("Request URI: %s, Method: %s", r.RequestURI, r.Method)

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
    case "surface":
        log.Println(fmt.Sprintf("We got a surface job: %+v", v))
        surfaceJob(w, v)
    case "squaring":
        log.Println(fmt.Sprintf("We got a squaring job: %+v", v))
        squaringJob(w, v)
    case "pocket":
        log.Println(fmt.Sprintf("We got a pocket job: %+v", v))
        pocketJob(w, v)
    case "circle":
        log.Println(fmt.Sprintf("We got a circle job: %+v", v))
        circleJob(w, v)
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

    var args = map[string]float64 {
        "width": v.Width,
        "length": v.Length,
        "depth": v.Depth,
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

    var args = map[string]float64 {
        "diameter": v.Diameter,
        "depth": v.Depth,
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

func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log the request URI and method
        log.Printf("Request URI: %s, Method: %s", r.RequestURI, r.Method)
        // Call the next handler
        next.ServeHTTP(w, r)
    })
}

//go:embed static
var content embed.FS
func runServer(port int) {
    sub, err := fs.Sub(content, "static")
    if err != nil {
        log.Fatal(err)
    }

    fileserver := http.FileServerFS(sub)

    http.HandleFunc("/api/dialog-result", apiHandler)
    http.Handle("/", Logger(fileserver))

    p := fmt.Sprintf(":%d", port)
    log.Printf("turtlecnc running on port: %d\n", port)
    if err := http.ListenAndServe(p, nil); err != nil{
        log.Fatal(err)
    }
}

