package main

import (
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	//	"strings"
)

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", serveTemplate)

	print("Listening on :9001...")
	err := http.ListenAndServe(":9001", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	print("förstöker serveraa " + url + "\n")
	lp := filepath.Join("templates", "layout.html")
	fp := filepath.Join("static", filepath.Clean(url))

	// First try to serve from static folder
	info, err := os.Stat(fp)
	if err == nil {
		// Serve static file
		if !info.IsDir() {
			print("Okje hittade statisk\n")
			http.ServeFile(w, r, fp)
		} else {
			// If static file does not exist try templates folder
			tp := filepath.Join("templates", filepath.Clean(url), "index.html")
			_, err := os.Stat(tp)
			if err == nil {
				tmpl, err := template.ParseFiles(lp, tp)
				if err != nil {
					serveNotFound(w, r)
				} else {
					print("serverar templatead fil\n")
					tmpl.ExecuteTemplate(w, "layout", nil)
				}
			} else {
				if os.IsNotExist(err) {
					// Try to serve directory contents
					print("serverar som vanlig fil\n")
					http.ServeFile(w, r, fp)
				}

			}
		}
	} else {
		if os.IsNotExist(err) {
			print("hittade inte " + fp + "\n")
			serveNotFound(w, r)
		} else {
			serveInternalError(w, r)
		}
	}
}

func serveNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	http.ServeFile(w, r, filepath.Join("templates", "404.html"))
}

func serveInternalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	http.ServeFile(w, r, filepath.Join("templates", "error.html"))
}

func removeElementAt(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

/*fs := http.FileServer(http.Dir("public/"))
http.Handle("/", fs)

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
})*/
