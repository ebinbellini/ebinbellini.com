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
	"regexp"
	"strings"
	texttemplate "text/template"
	"time"
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

	// Used in RSS
	PubDate    string
	Desc       string
	LastChange string
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

	SinglePost bool

	FoundDocuments []DocumentMatch
}

type FeedData struct {
	LastPostTime   string
	LastChangeTime string
	Posts          []BlogPost
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", serveTemplate)
	http.HandleFunc("/blog/", serveBlogPage)
	http.HandleFunc("/blog/post/", serveBlogPost)
	http.HandleFunc("/knaker/", redirectToKnaker)
	http.HandleFunc("/query/", serveQuery)
	http.HandleFunc("/rss/", serveRSS)

	fmt.Println("Listening on :9001...")
	err := http.ListenAndServe(":9001", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
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
				http.ServeFile(w, r, fp)
			} else {
				// If static file does not exist try templates folder
				tp := filepath.Join("templates", filepath.Clean(url), "index.html")
				_, err := os.Stat(tp)
				if err == nil {
					// Add a / to the end of the URL if there isn't on already
					if !strings.HasSuffix(url, "/") {
						http.Redirect(w, r, url+"/", http.StatusMovedPermanently)
						return
					}

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
				print("Couldn't find " + fp + "\n")
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
		return
	}

	found := []DocumentMatch{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			serveInternalError(w, r)
		}
		defer f.Close()

		parts := strings.Split(strings.Replace(file, "\\", "/", -1), "/")
		title := strings.Title(parts[len(parts)-2])

		matching := []string{}

		// Match against text content
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanWords)
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

		// Match against title
		for _, str := range search {
			if strings.Contains(strings.ToLower(title), strings.ToLower(str)) {
				matching = append(matching, title)
				break
			}
		}

		// Clean up matches
		for i, match := range matching {
			re := regexp.MustCompile(`(.*\=\")|(\/\"\>$)|(\"\/\>$)|(\<\/.*\>$)|(</.*>)|("\>)|(,)`)
			matching[i] = strings.TrimSpace(string(re.ReplaceAll([]byte(match), []byte(" "))))
		}

		if len(matching) > 0 {
			if title == "Templates" {
				title = "Home"
			}
			path := strings.TrimRight(strings.TrimLeft(strings.Replace(file, "\\", "/", -1), "templates\\"), ".index.html")
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

func serveRSS(w http.ResponseWriter, r *http.Request) {
	posts := blogPosts(w, r)
	if posts == nil {
		serveNotFound(w, r)
		return
	}

	last := posts[len(posts)-1]
	data := FeedData{
		LastPostTime:   last.PubDate,
		LastChangeTime: last.LastChange,
		Posts:          posts,
	}

	tmpl := texttemplate.Must(texttemplate.ParseFiles("rss.xml"))
	tmpl.ExecuteTemplate(w, "RSS", data)
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
		return
	}

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

	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		serveInternalError(w, r)
	}
}

func serveBlogPost(w http.ResponseWriter, r *http.Request) {
	location := filepath.Join("static", strings.ReplaceAll(strings.TrimLeft(r.URL.Path, "/"), "/post", ""))

	lp := filepath.Join("templates", "layout.html")
	tp := filepath.Join("templates", "blog", "index.html")
	tmpl, err := template.ParseFiles(lp, tp)
	if err != nil {
		serveNotFound(w, r)
		return
	}

	info, err := os.Stat(location)
	if err != nil {
		serveNotFound(w, r)
		return
	}

	post, err := getBlogPostData(location, info)
	if err != nil {
		serveInternalError(w, r)
		return
	}

	data := PageData{
		BlogPosts:  []BlogPost{post},
		SinglePost: true,
	}

	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		serveInternalError(w, r)
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

		// Skip the static/blog folder
		baseFolder := filepath.Join("static", "blog")
		if path == baseFolder {
			return nil
		}

		post, err := getBlogPostData(path, info)
		if err != nil {
			fmt.Println(err)
			serveInternalError(w, r)
		}
		posts = append(posts, post)

		return nil
	})
	if err != nil {
		serveInternalError(w, r)
	}

	return posts
}

func getBlogPostData(path string, info os.FileInfo) (BlogPost, error) {
	name := info.Name()
	contentPath := filepath.Join(path, "index.html")

	file, err := os.Open(contentPath)
	if err != nil {
		return BlogPost{}, err
	}
	defer file.Close()

	// Read entire blog post file
	buf := new(strings.Builder)
	_, err = io.Copy(buf, file)
	if err != nil {
		return BlogPost{}, err
	}

	// Create a sneak peak of the content
	desc := buf.String()
	// Remove HTML tags, tabs, and carriage returns
	re := regexp.MustCompile(`(<div .*</div>)|(<.*>)|(</.*>)|(<.*/>)|(\t+)|(\r)`)
	desc = strings.TrimSpace(string(re.ReplaceAll([]byte(desc), []byte(""))))
	// Remove new lines (put everything on one line)
	re = regexp.MustCompile(`(\n)`)
	desc = string(re.ReplaceAll([]byte(desc), []byte(" ")))
	// Limit the string to 150 bytes
	if len(desc) > 150 {
		desc = strings.TrimSpace(desc[0:146]) + "..."
	}

	// Time format for XML
	const rfc2822 = "Mon Jan 02 15:04:05 -0700 2006"
	const titleformat = "2006-01-02"

	datePublished, err := time.Parse(titleformat, name)
	if err != nil {
		return BlogPost{}, err
	}
	pubDate := datePublished.Format(rfc2822)

	return BlogPost{
		Name: name,
		// TODO Make pages for individual posts
		Path:       "https://ebinbellini.top/blog/post/" + name + "/",
		Desc:       desc,
		Content:    template.HTML(buf.String()),
		Tags:       extractBlogPostTags(contentPath),
		PubDate:    pubDate,
		LastChange: info.ModTime().Format(rfc2822),
	}, nil
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

func redirectToKnaker(w http.ResponseWriter, r *http.Request) {
	// Add a / to the end of the URL if there isn't on already
	url := r.URL.Path
	suffix := ""
	if !strings.HasSuffix(url, "/") {
		suffix = "/"
	}

	// Redirect to the correct URL
	http.Redirect(w, r, "/works"+r.URL.Path+suffix, http.StatusMovedPermanently)
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
			title := strings.Title(strings.Replace(strings.TrimSuffix(strings.TrimPrefix(url, "/works/"), "/"), "/", " > ", -1))
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
