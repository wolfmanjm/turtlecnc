package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"runtime"

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

// Middleware handler to log requests
type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    log.Printf("Request URI: %s, Method: %s", r.RequestURI, r.Method)
	l.handler.ServeHTTP(w, r)
}

// NewLogger constructs a new Logger middleware handler
func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

func panicRecovery(h func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                buf := make([]byte, 2048)
                n := runtime.Stack(buf, false)
                buf = buf[:n]

                log.Printf("recovering from panic: %v\n %s", err, buf)
                w.Write([]byte(`{"error":"Internal server error, please report what you were doing"}`))
            }
        }()

        h(w, r)
    }
}

//go:embed static
var content embed.FS
func runServer(port int) {
    sub, err := fs.Sub(content, "static")
    if err != nil {
        log.Fatal(err)
    }

	mux := http.NewServeMux()

    fileserver := http.FileServerFS(sub)

    mux.HandleFunc("/api/dialog-result", panicRecovery(apiHandler))
    mux.Handle("/", fileserver)

	muxWithLogger := NewLogger(mux)
    p := fmt.Sprintf(":%d", port)
    log.Printf("turtlecnc running on port: %d\n", port)
    if err := http.ListenAndServe(p, muxWithLogger); err != nil{
        log.Fatal(err)
    }
}

