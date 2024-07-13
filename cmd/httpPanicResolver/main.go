package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
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
				// write 500 in case of any panic
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "<h2>Panic: %v</h2><pre>%s</pre>", err, string(errorStack))
			}
		}()
		//to rewrite the partial response, make a copy of responseWriter
		nwr := &newResponseWriter{ResponseWriter: w}
		muxHandler.ServeHTTP(nwr, r)
		nwr.flush() // O(n^2) extra overhead by using wrapper
	}
}

func createMultiplex() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePageHandler)
	mux.HandleFunc("/panic", PanicHandler)
	mux.HandleFunc("/panic-reset", ResetResponseHandler)
	return mux
}

// A wrapper for http.responseWriter
// cons - stores all the write buffer in memory instead of streaming it to client as the http.ResponseWriter does
// Solution - See https://pkg.go.dev/net/http#Flusher
type newResponseWriter struct {
	http.ResponseWriter
	writes [][]byte
	status int
}

func (nwr *newResponseWriter) Write(b []byte) (int, error) {
	// TODO - flush writes to the client once writes exceed some thresh hold, then only write to the writes
	// to prevent memory overload
	nwr.writes = append(nwr.writes, b)
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
