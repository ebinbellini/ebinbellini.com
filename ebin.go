package main

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type CollectionLink struct {
	Name     string
	Path     string
	Image    string
	ImgCount int
}

type BlogPost struct {
	Name    string
	Path    string
	Content template.HTML
	Tags    []string
}

type DocumentMatch struct {
	Name          string
	Path          string
	MatchingWords string
}

type PageData struct {
	Links       []CollectionLink
	BlogPosts   []BlogPost
	SelectedTag string

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
	http.HandleFunc("/blog/", serveBlogPage)

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

	if strings.HasPrefix(url, "/works/gallery/") && !strings.Contains(url, ".") {
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
				{
					Name:          "",
					Path:          "#",
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

func stringArrayHas(array []string, target string) bool {
	for _, s := range array {
		if s == target {
			return true
		}
	}

	return false
}

func serveBlogPage(w http.ResponseWriter, r *http.Request) {
	lp := filepath.Join("templates", "layout.html")
	tp := filepath.Join("templates", "blog", "index.html")
	tmpl, err := template.ParseFiles(lp, tp)
	if err != nil {
		serveNotFound(w, r)
	} else {
		posts := blogPosts(w, r)
		if posts == nil {
			serveNotFound(w, r)
			return
		}

		data := PageData{}

		url := r.URL.Path
		filtered := []BlogPost{}
		tag := strings.Split(url, "/")[2]
		if tag != "" {
			data.SelectedTag = tag

			for _, post := range posts {
				if stringArrayHas(post.Tags, tag) {
					filtered = append(filtered, post)
				}
			}

			posts = filtered
		}

		// Reverse posts
		for i, j := 0, len(posts)-1; i < j; i, j = i+1, j-1 {
			posts[i], posts[j] = posts[j], posts[i]
		}

		data.BlogPosts = posts

		fmt.Println("DEN VALDA TAGGEN ÄR.... ", data.SelectedTag)

		err := tmpl.ExecuteTemplate(w, "layout", data)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func blogPosts(w http.ResponseWriter, r *http.Request) []BlogPost {
	fp := filepath.Join("static", "blog")

	// Gather all blog posts in an array
	posts := []BlogPost{}

	// Walk through all files in blog folder
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		// Only interesteed in folders
		if !info.IsDir() {
			return nil
		}

		name := info.Name()
		contentPath := filepath.Join(path, "index.html")
		file, err := os.Open(contentPath)
		if err != nil {
			return nil
		}
		defer file.Close()

		buf := new(strings.Builder)
		_, err = io.Copy(buf, file)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		// Store this post
		posts = append(posts, BlogPost{
			Name:    name,
			Path:    name + "/",
			Content: template.HTML(buf.String()),
			Tags:    extractBlogPostTags(contentPath),
		})

		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}

	return posts
}

func extractBlogPostTags(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	tags := []string{}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return tags
	}

	// Skip looking for tags if the file does not start with a tag container
	if !strings.Contains(line, `<div class="tags">`) {
		return tags
	}

	for {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}

		// Return at the end of the tag container
		if strings.Contains(line, `</div>`) {
			return tags
		}

		// Get tag from within quotation marks
		tags = append(tags, strings.Split(line, `"`)[1])
	}

	return tags
}

func serveGalleryPage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	lp := filepath.Join("templates", "layout.html")
	if url == "/works/gallery/" {
		tp := filepath.Join("templates", "works", "gallery", "index.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			links := imageGalleryCollectionLinks(w, r)
			data := PageData{
				Links: links,
			}
			err := tmpl.ExecuteTemplate(w, "layout", data)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		tp := filepath.Join("templates", "works", "gallery", "template.html")
		tmpl, err := template.ParseFiles(lp, tp)
		if err != nil {
			serveNotFound(w, r)
		} else {
			title := strings.Title(strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(url, "/works/"), "/"), "/", " > "))
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

func imageGalleryCollectionLinks(w http.ResponseWriter, r *http.Request) []CollectionLink {
	fp := filepath.Join("static", "works", "gallery")
	links := []CollectionLink{}
	err := filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		file, err := os.Open(path)
		images, _ := file.Readdirnames(0)
		defer file.Close()

		if !strings.Contains(name, ".") && name != "gallery" {
			links = append(links, CollectionLink{
				Name:     strings.Title(name),
				Path:     name + "/",
				Image:    images[0],
				ImgCount: len(images),
			})
		}
		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}
	return links
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
