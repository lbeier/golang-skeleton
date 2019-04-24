package main

import (
	"github.com/tutabeier/golang-skeleton/pkg"
	"net/http"
)
import "github.com/gorilla/mux"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("up and running"))
	})
	if err := http.ListenAndServe(":9999", nil); err != nil {
		panic(err)
	}
}

func routes()  {
	r := mux.NewRouter()
	r.HandleFunc("/status", pkg.Status())
}