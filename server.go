package main

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type CollectionLink struct {
	Name string
	Path string
}

type DocumentMatch struct {
	Name          string
	Path          string
	MatchingWords string
}

type PageData struct {
	Links []CollectionLink

	Title          string
	ImageColumnOne []string
	ImageColumnTwo []string

	FoundDocuments []DocumentMatch
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/query/", serveQuery)
	http.HandleFunc("/", serveTemplate)

	fmt.Println("Listening on :9001...")
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

	if strings.HasPrefix(url, "/gallery/") && !strings.Contains(url, ".") {
		serveGalleryPage(w, r)
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

func serveQuery(w http.ResponseWriter, r *http.Request) {
	searchQueryValue := r.FormValue("s")
	search := strings.Split(searchQueryValue, " ")

	fp := "templates"
	files := []string{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "index.html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}

	found := []DocumentMatch{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			serveInternalError(w, r)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanWords)

		matching := []string{}
		for scanner.Scan() {
			text := scanner.Text()
			if strings.Contains(text, "{") || strings.Contains(text, "}") {
				continue
			}
			for _, str := range search {
				if strings.Contains(strings.ToLower(text), strings.ToLower(str)) {
					matching = append(matching, text)
					break
				}
			}
		}

		if len(matching) > 0 {
			parts := strings.Split(strings.ReplaceAll(file, "\\", "/"), "/")
			title := strings.Title(parts[len(parts)-2])
			if title == "Templates" {
				title = "Home"
			}
			path := strings.TrimRight(strings.TrimLeft(strings.ReplaceAll(file, "\\", "/"), "templates\\"), ".index.html")
			found = append(found, DocumentMatch{
				Name:          title,
				Path:          path,
				MatchingWords: "Contains: " + strings.Join(matching, ", "),
			})
		}
	}

	data := PageData{
		FoundDocuments: found,
	}

	if len(data.FoundDocuments) == 0 {
		data = PageData{
			FoundDocuments: []DocumentMatch{
				DocumentMatch{
					Name: "",
					Path: "#",
					MatchingWords: "No results found for \"" + searchQueryValue + "\".",
				},
			},
		}
	}

	lp := filepath.Join("templates", "layout.html")
	tp := filepath.Join("templates", "query", "index.html")
	tmpl, err := template.ParseFiles(lp, tp)
	if err != nil {
		fmt.Println(err)
	}
	tmpl.ExecuteTemplate(w, "layout", data)
}

func serveGalleryPage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	lp := filepath.Join("templates", "layout.html")
	if url == "/gallery/" {
		tp := filepath.Join("templates", "gallery", "index.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			collections := listImageGalleryCollections(w, r)
			links := []CollectionLink{}
			for _, name := range collections {
				links = append(links, CollectionLink{
					Name: strings.Title(name),
					Path: name + "/",
				})
			}
			data := PageData{
				Links: links,
			}
			err := tmpl.ExecuteTemplate(w, "layout", data)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		tp := filepath.Join("templates", "gallery", "template.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			title := strings.Title(strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(url, "/"), "/"), "/", " > "))
			images, err := listGalleryImages(w, r)
			if err != nil {
				serveNotFound(w, r)
				return
			}
			columnOne := images[:len(images)/2]
			columnTwo := images[len(images)/2:]
			data := PageData{
				Title:          title,
				ImageColumnOne: columnOne,
				ImageColumnTwo: columnTwo,
			}
			err = tmpl.ExecuteTemplate(w, "layout", data)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func listImageGalleryCollections(w http.ResponseWriter, r *http.Request) []string {
	fp := filepath.Join("static", "gallery")
	folders := []string{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		if !strings.Contains(name, ".") && name != "gallery" {
			folders = append(folders, name)
		}
		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}
	return folders
}

func listGalleryImages(w http.ResponseWriter, r *http.Request) ([]string, error) {
	url := r.URL.Path
	fp := filepath.Join("static", url)
	files := []string{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(info.Name(), ".") {
			files = append(files, info.Name())
		}
		return nil
	})
	if err != nil {
		return nil, errors.New("Image gallery collection " + fp + " not found")
	}

	return files, nil
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
