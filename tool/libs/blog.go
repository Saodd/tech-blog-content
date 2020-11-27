package libs

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"regexp"
)

const PublicBlogDir = "./public"
const blogInputExpr = "(?s)^```yaml lw-blog-meta(.*?)```"
const blogOutputPref = "```json lw-blog-meta\n"
const blogExprPrefLen = 20
const blogExprPostLen = 3
const blogPicLocal = `../../../../tech-blog-pic/`
const blogPicCloud = `https://cdn.jsdelivr.net/gh/Saodd/tech-blog-pic@gh-pages/`

var blogPattern, _ = regexp.Compile(blogInputExpr)

type Blog struct {
	Title string   `yaml:"title" json:"title"`
	Date  string   `yaml:"date" json:"date"`
	Brev  string   `yaml:"brev" json:"brev"`
	Tags  []string `yaml:"tags" json:"tags"`
	Path  string   `json:"path"`
	Body  []byte   `json:"-"`
}

func (b *Blog) PublicDirname() string {
	return path.Join(PublicBlogDir, path.Dir(b.Path))
}

func (b *Blog) PublicWrite() error {
	j, err := json.Marshal(b)
	if err != nil {
		return err
	}
	b.Body = bytes.ReplaceAll(b.Body, []byte(blogPicLocal), []byte(blogPicCloud))
	buf := []byte(blogOutputPref)
	buf = append(buf, j...)
	buf = append(buf, "\n```\n"...)
	buf = append(buf, b.Body...)
	return ioutil.WriteFile(path.Join(PublicBlogDir, b.Path), buf, 0755)
}

func calcHashPath(filePath string, text []byte) string {
	m := fmt.Sprintf("%x", md5.Sum(text))[:20]
	filename := path.Base(filePath) + "." + m
	return path.Join(path.Dir(filePath), filename)
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
		blog.Path = calcHashPath(p, text)
		blogs = append(blogs, blog)
	}
	return blogs, nil
}

func parseBlogFile(text []byte) (*Blog, error) {
	meta := blogPattern.Find(text)
	if len(meta) <= blogExprPrefLen {
		return nil, errors.New("头部信息无效。")
	}

	var blog = &Blog{}
	if err := yaml.Unmarshal(meta[blogExprPrefLen:len(meta)-blogExprPostLen], blog); err != nil {
		return nil, err
	}
	blog.Body = text[len(meta):]

	return blog, nil
}

func SaveBlogs(blogs []*Blog) error {
	var subDirs = map[string]bool{}
	for _, blog := range blogs {
		subDir := blog.PublicDirname()
		// 确保输出文件夹
		if !subDirs[subDir] {
			if err := os.MkdirAll(subDir, 0755); err != nil {
				return err
			}
			subDirs[subDir] = true
		}
		// 输出文件
		if err := blog.PublicWrite(); err != nil {
			return err
		}
	}
	return nil
}
