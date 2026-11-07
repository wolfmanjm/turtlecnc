package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"

	"io/fs"
	"log"
	"net/http"

	cams "github.com/wolfmanjm/turtlecnc/internal"
	tui "github.com/wolfmanjm/turtlecnc/tui"
)

func main() {
    // get flags
    test := flag.Bool("t", false, "debug test")
    server := flag.Bool("s", false, "start server")
    port := flag.Int("port", 8080, "Set server listen port")
    runTui := flag.Bool("tui", false, "run tui interface")
    debug := flag.Bool("debug", false, "run in debug mode")

    flag.Parse()

    if *test {
	   cams.Testit()
    } else if *server {
        runServer(*port)
    } else if *runTui {
    	tui.Debug = *debug
    	tui.RunTui()
    } else {
        flag.PrintDefaults()
    }
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

