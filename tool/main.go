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
	libs.SyncServer(blogs)
}
