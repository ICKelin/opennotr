package main

import "net/http"

func main() {
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.RemoteAddr))
	})
	http.ListenAndServe(":8080", nil)
}
