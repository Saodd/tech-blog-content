package main

import (
	"log"
	"tech-blog-content/tool/libs"
)

func main() {
	libs.CheckWorkDir()
	files := libs.RecurListMds("./blog")
	blogs, err := libs.ParseBlogFiles(files)
	if err != nil {
		log.Fatalln(err)
	}
	if err := libs.SaveBlogs(blogs); err != nil {
		log.Fatalln(err)
	}
	if err := libs.GenIndexes(blogs); err != nil {
		log.Fatalln(err)
	}
}
