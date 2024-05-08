package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := createMultiplex()
	log.Fatal(http.ListenAndServe(":7020", recoverHttpMiddleware(mux)))
}

func recoverHttpMiddleware(muxHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
		}()
		muxHandler.ServeHTTP(w, r)
	}
}

func createMultiplex() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePageHandler)
	mux.HandleFunc("/panic", PanicHandler)
	mux.HandleFunc("/panic-reset", ResetResponseHandler)
	return mux
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
