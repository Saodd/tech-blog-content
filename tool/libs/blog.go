package libs

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"regexp"
)

const blogInputExpr = "(?s)^```yaml lw-blog-meta(.*?)```"
const blogExprPrefLen = 20
const blogExprPostLen = 3
const blogPicLocal = `](../pic/`
const blogPicCloud = `](https://gcore.jsdelivr.net/gh/saodd/tech-blog-content@gh-pages/pic/`
const blogMarkdownLocal = `](../20`
const blogMarkdownCloud = `](/blog/page/20`

var blogPattern, _ = regexp.Compile(blogInputExpr)

type Blog struct {
	Title       string   `yaml:"title" json:"title"`
	Date        string   `yaml:"date" json:"date"`
	Brev        string   `yaml:"brev" json:"brev"`
	Tags        []string `yaml:"tags" json:"tags"`
	Description string   `yaml:"description" json:"description"`
	Keywords    string   `yaml:"keywords" json:"keywords"`
	Path        string   `json:"path"`
	Body        string   `json:"body"`
	Hash        string   `json:"hash"`
}

func ParseBlogFiles(filePaths []string) (blogs []*Blog, err error) {
	for _, p := range filePaths {
		text, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}
		blog, err := parseBlogFile(text)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("解析失败(%s): %s\n", p, err))
		}
		blog.Path = p
		blogs = append(blogs, blog)
	}
	return blogs, nil
}

func parseBlogFile(text []byte) (*Blog, error) {
	text = bytes.ReplaceAll(text, []byte("\r\n"), []byte("\n"))
	text = bytes.ReplaceAll(text, []byte(blogPicLocal), []byte(blogPicCloud))
	text = bytes.ReplaceAll(text, []byte(blogMarkdownLocal), []byte(blogMarkdownCloud))
	meta := blogPattern.Find(text)
	if len(meta) <= blogExprPrefLen {
		return nil, errors.New("头部信息无效。")
	}

	var blog = &Blog{}
	if err := yaml.Unmarshal(meta[blogExprPrefLen:len(meta)-blogExprPostLen], blog); err != nil {
		return nil, err
	}

	blog.Body = string(text[len(meta):])
	blog.Hash = fmt.Sprintf("%x", md5.Sum(text))

	return blog, nil
}
