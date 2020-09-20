package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TODO search using a form without javascript

type CollectionLink struct {
	Name string
	Path string
}

type CollectionPageData struct {
	Links []CollectionLink

	Title          string
	ImageColumnOne []string
	ImageColumnTwo []string
}

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

	if strings.HasPrefix(url, "/collections/") && !strings.Contains(url, ".") {
		serveCollectionsPage(w, r)
	} else {
		// First try to serve from static folder
		info, err := os.Stat(fp)
		if err == nil {
			// Serve static file
			if !info.IsDir() {
				fmt.Println("static file found", fp)
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
						tmpl.ExecuteTemplate(w, "layout", nil)
					}
				} else {
					if os.IsNotExist(err) {
						// Try to serve directory contents
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
}

func serveCollectionsPage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	lp := filepath.Join("templates", "layout.html")
	if url == "/collections/" {
		tp := filepath.Join("templates", "collections", "index.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			collections := listImageCollections(w, r)
			links := []CollectionLink{}
			for _, name := range collections {
				links = append(links, CollectionLink{
					Name: strings.Title(name),
					Path: name + "/",
				})
			}
			data := CollectionPageData{
				Links: links,
			}
			err := tmpl.ExecuteTemplate(w, "layout", data)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		tp := filepath.Join("templates", "collections", "template.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			title := strings.Title(strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(url, "/"), "/"), "/", " > "))
			images := listCollectionImages(w, r)
			columnOne := images[:len(images)/2]
			columnTwo := images[len(images)/2:]
			data := CollectionPageData{
				Title:          title,
				ImageColumnOne: columnOne,
				ImageColumnTwo: columnTwo,
			}
			err := tmpl.ExecuteTemplate(w, "layout", data)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func listImageCollections(w http.ResponseWriter, r *http.Request) []string {
	fp := filepath.Join("static", "collections")
	folders := []string{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		if !strings.Contains(name, ".") && name != "collections" {
			folders = append(folders, name)
		}
		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}
	return folders
}

func listCollectionImages(w http.ResponseWriter, r *http.Request) []string {
	url := r.URL.Path
	fp := filepath.Join("static", url)
	files := []string{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(info.Name(), ".") {
			files = append(files, info.Name())
		}
		return nil
	})
	if err != nil {
		serveNotFound(w, r)
	}

	return files
}

func serveNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	http.ServeFile(w, r, filepath.Join("templates", "404.html"))
}

func serveInternalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	http.ServeFile(w, r, filepath.Join("templates", "error.html"))
}

/*func removeElementAt(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}*/
