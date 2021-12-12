package main

import (
	"context"
	"github.com/saodd/alog"
	"os"
	"tech-blog-content/tool/libs"
	"time"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (err error) {
	c, cancel := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancel()
	ctx, cancel := alog.WithTracker(c)
	defer cancel()
	defer alog.CERecoverError(ctx, &err)

	if err := libs.CheckWorkDir(ctx); err != nil {
		alog.CE(ctx, err)
		return err
	}
	files, err := libs.RecurListMds(ctx, ".")
	if err != nil {
		alog.CE(ctx, err)
		return err
	}
	blogs, err := libs.ParseBlogFiles(files)
	if err != nil {
		alog.CE(ctx, err)
		return err
	}
	return libs.SyncServer(ctx, blogs)
}
