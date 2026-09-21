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

    w.Header().Set("Content-Type", "application/text")

    // handle each job kind
    switch kind {
    case "rectangle":
        log.Println(fmt.Sprintf("We got a rectangle job, values: %v", values))
        rectangleJob(w, values)
    case "hole":
        log.Println("We got a hole job", values)
        holeJob(w, values)
    case "circle":
        log.Println("We got a circle job", values)
    default:
        log.Println("Unknown kind: ", kind)
        fmt.Fprintf(w, "Unknown kind: ", kind)
        return
    }
}

func rectangleJob(w io.Writer, v map[string]any) {
    // depth:6 doc:1.5 feedrate:1400 jobname:rectangle length:150 rpm:10000 side:laser tooldia:6.35 width:100
    g := cams.NewCam(w)

    g.Laser = false

    inside := true
    switch v["side"] {
    case "inside":
        inside = true
    case "outside":
        inside = false
    case "laser":
        g.Laser = true
    default:
        fmt.Fprintf(w, "Unknown side: %v", v["side"])
        return
    }

    g.Tool_diameter = v["tooldia"].(float64)
    g.Doc = v["doc"].(float64)
    g.RPM = v["rpm"].(float64)

    var args = map[string]float64 {
        "width": v["width"].(float64),
        "length": v["length"].(float64),
        "depth": v["depth"].(float64),
        "feedrate": v["feedrate"].(float64),
    }

     if err := g.Rectangle(inside, args); err != nil {
        fmt.Fprintf(w, "Rectangle() failed: %w", err)
    }
}

func holeJob(w io.Writer, v map[string]any) {
    // depth:12 feedrate:1400 holedia:5 jobname:hole laser:false peck:0 rpm:10000 tooldia:6.35
    g := cams.NewCam(w)

    g.Laser = v["laser"].(bool)

    if !g.Laser {
        g.Tool_diameter = v["tooldia"].(float64)
        g.Doc = v["doc"].(float64)
        g.RPM = v["rpm"].(float64)
    }

    var args = map[string]float64 {
        "diameter": v["holedia"].(float64),
        "depth": v["depth"].(float64),
        "feedrate": v["feedrate"].(float64),
        "peck": v["peck"].(float64),
    }

     if err := g.Hole(args); err != nil {
        fmt.Fprintf(w, "Hole() failed: %v", err.Error())
    }
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

