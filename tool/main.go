package main

import (
	"log"
	"net/http"
	"os"
	"tech-blog-content/tool/libs"
)

func main() {
	if os.Getenv("RUN_MODE") == "gh-actions" {
		processMarkdown()
	} else {
		go processMarkdown()

		var orig = http.StripPrefix("/", http.FileServer(http.Dir("./public")))
		var wrapped = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			orig.ServeHTTP(w, r)
		})
		http.ListenAndServe(":7777", wrapped)
	}
}

func processMarkdown() {
	libs.CheckWorkDir()
	files := libs.RecurListMds(".")
	blogs, err := libs.ParseBlogFiles(files)
	if err != nil {
		log.Fatalln(err)
	}
	if err := libs.SaveBlogs(blogs); err != nil {
		log.Fatalln(err)
	}
	libs.QuickSortBlog(blogs)
	if err := libs.PostPathsToServer(blogs); err != nil {
		log.Fatalln(err)
	}
}
