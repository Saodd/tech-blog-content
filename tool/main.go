package main

import (
	"log"
	"tech-blog-content/tool/libs"
)

func main() {
	libs.CheckWorkDir()
	files := libs.RecurListMds(".")
	blogs, err := libs.ParseBlogFiles(files)
	if err != nil {
		log.Fatalln(err)
	}
	if err := libs.SaveBlogs(blogs); err != nil {
		log.Fatalln(err)
	}
	if err := libs.PostPathsToServer(blogs); err != nil {
		log.Fatalln(err)
	}
}
