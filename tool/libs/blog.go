package libs

import (
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
const blogExpr = "(?s)^```yaml lw-blog-meta(.*?)```"
const blogExprPrefLen = 20
const blogExprPostLen = 3

var blogPattern *regexp.Regexp

func init() {
	blogPattern, _ = regexp.Compile(blogExpr)
}

type Blog struct {
	Title string   `yaml:"title"`
	Date  string   `yaml:"date"`
	Brev  string   `yaml:"brev"`
	Tags  []string `yaml:"tags"`
	Path  string   `json:"-"`
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
	buf := []byte("```json lw-blog-meta\n")
	buf = append(buf, j...)
	buf = append(buf, "\n```\n"...)
	buf = append(buf, b.Body...)
	return ioutil.WriteFile(path.Join(PublicBlogDir, b.Path), buf, 0755)
}

func ParseBlogFiles(filePaths []string) (blogs []*Blog, err error) {
	for _, p := range filePaths {
		blog, err := parseBlogFile(p)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("解析失败(%s): %s\n", p, err))
		}
		blogs = append(blogs, blog)
	}
	return blogs, nil
}

func parseBlogFile(filePath string) (*Blog, error) {
	text, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	meta := blogPattern.Find(text)
	if len(meta) <= blogExprPrefLen {
		return nil, errors.New("头部信息无效。")
	}

	var blog = &Blog{}
	if err := yaml.Unmarshal(meta[blogExprPrefLen:len(meta)-blogExprPostLen], blog); err != nil {
		return nil, err
	}
	blog.Path = filePath
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
