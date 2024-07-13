package main

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
)

func main() {
	mux := createMultiplex()
	log.Fatal(http.ListenAndServe(":7020", recoverHttpMiddleware(mux, true)))
}

func recoverHttpMiddleware(muxHandler http.Handler, isDev bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
				errorStack := debug.Stack() // returns a formatted stack trace of the goroutines that calls it
				log.Println(string(errorStack))
				if !isDev {
					http.Error(w, "Something went wrong", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				// write 500 in case of any panic
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "<h2>Panic: %v</h2>\n<pre>%s</pre>", err, makeLinks(string(errorStack)))
			}
		}()
		//to rewrite the partial response, make a copy of responseWriter
		nwr := &newResponseWriter{ResponseWriter: w}
		muxHandler.ServeHTTP(nwr, r)
		nwr.flush()

	}
}

func createMultiplex() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePageHandler)
	mux.HandleFunc("/panic", PanicHandler)
	mux.HandleFunc("/panic-reset", ResetResponseHandler)
	mux.HandleFunc("/debug/", SourceCodeHandler)
	return mux
}

// A wrapper for http.responseWriter
// cons - stores all the write buffer in memory instead of streaming it to client as the http.ResponseWriter does
type newResponseWriter struct {
	http.ResponseWriter
	writes [][]byte
	status int
}

func (nwr *newResponseWriter) Write(b []byte) (int, error) {
	c := make([]byte, len(b))
	copy(c, b) // see https://github.com/gophercises/recover_chroma/issues/1
	nwr.writes = append(nwr.writes, c)
	return len(b), nil
}
func (nwr *newResponseWriter) WriteHeader(status int) {
	nwr.status = status
}
func (nwr *newResponseWriter) flush() error {
	if nwr.status != 0 {
		nwr.ResponseWriter.WriteHeader(nwr.status)
	}
	for _, write := range nwr.writes {
		_, err := nwr.ResponseWriter.Write(write)
		if err != nil {
			return err
		}
	}
	return nil
}

func HomePageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	_, err := responseWriter.Write([]byte("Welcome to the home page!"))
	if err != nil {
		return
	}
}
func PanicHandler(responseWriter http.ResponseWriter, request *http.Request) {
	panickingFunctions()
}
func ResetResponseHandler(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprint(responseWriter, "<h1>Partial Response Write</h1>")
	panickingFunctions()
}
func panickingFunctions() {
	panic("I'm panicking")
}
func SourceCodeHandler(responseWriter http.ResponseWriter, request *http.Request) {
	path := request.FormValue("source")
	lineNumber := request.FormValue("line")
	line, err := strconv.Atoi(lineNumber)
	if err != nil {
		line = -1
	}
	file, err := os.Open(path)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	b := bytes.NewBuffer(nil)
	_, err = io.Copy(b, file)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	var lineRange [][2]int
	if line > 0 {
		lineRange = append(lineRange, [2]int{line, line})
	}
	formatter := html.New(
		html.TabWidth(2),
		html.WithLineNumbers(true),
		html.LineNumbersInTable(true),
		html.WithLinkableLineNumbers(true, ""),
		html.HighlightLines(lineRange),
	)
	lex := lexers.Get("go")
	iter, err := lex.Tokenise(nil, b.String())
	if err != nil {
		log.Println(err)
	}
	style := styles.Get("pastie") // https://swapoff.org/chroma/playground/
	if style == nil {
		style = styles.Fallback
	}
	fmt.Fprint(responseWriter, "<style>pre {front-size: 1.2em}</style>")
	_ = formatter.Format(responseWriter, style, iter)
	//_ = quick.Highlight(responseWriter, b.String(), "go", "html", "monokai")
}
