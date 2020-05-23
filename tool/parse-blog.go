package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
)

const PublicBlogDir = "./public/blog"
const PublicIndexDir = "./public/index"

func main() {
	checkWorkDir()
	recurOnDir("./blog")
	GenIndexes()
}

func checkWorkDir() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	if path.Base(workDir) != "tech-blog-content" {
		log.Fatalln("当前路径不是项目根目录(tech-blog-content)！")
	}
}

func recurOnDir(folder string) {
	files, _ := ioutil.ReadDir(folder)
	for _, file := range files {
		if file.IsDir() {
			recurOnDir(path.Join(folder, file.Name()))
		} else {
			if name := file.Name(); len(name) > 3 && name[len(name)-3:] == ".md" {
				ParseMdFile(path.Join(folder, file.Name()))
			}
		}
	}

}

// ParseMdFile 负责解析指定的md文件，提取结构化的数据。并交给 processBlog 进行下一步处理。
func ParseMdFile(filePath string) {
	text, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println(err)
		return
	}

	meta := metaPattern.Find(text)
	if len(meta) <= 20 {
		log.Println("该文件的头部描述信息有误:", filePath)
		return
	}

	var blog = &Blog{}
	if err := json.Unmarshal(meta[15:len(meta)-3], blog); err != nil {
		log.Printf("解析失败(%s): %s\n", filePath, err)
		return
	}
	blog.Path = filePath
	raw := text[len(meta):]

	processBlog(blog, raw)
}

func processBlog(blog *Blog, raw []byte) {
	// 无论是否能输出到指定目录，都先保存索引
	blogs = append(blogs, blog)
	// 确保输出文件夹
	err := os.MkdirAll(path.Join(PublicBlogDir, path.Dir(blog.Path)), 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 输出文件
	err = ioutil.WriteFile(path.Join(PublicBlogDir, blog.Path), raw, 0755)
	if err != nil {
		fmt.Println(err)
	}
}

func GenIndexes() {
	QuickSortBlog(blogs)
	// 根据各个文章的tag以及一个额外的 TimeLine，生成相应的索引列表。
	tagMap := make(map[string][]*Blog)
	tagMap["TimeLine"] = blogs
	for _, blog := range blogs {
		for _, tag := range blog.Tags {
			tagMap[tag] = append(tagMap[tag], blog)
		}
	}
	// 把所有的索引导出为文件
	err := os.MkdirAll(PublicIndexDir, 0755)
	if err != nil {
		log.Println(err)
		return
	}
	if err := GenRootIndex(tagMap); err != nil {
		log.Println(err)
	}
	GenTagIndex(tagMap)
}

// GenRootIndex 生成一个总索引。形式是 [标签名:文章数]，用于左侧导航栏。
func GenRootIndex(tagMap map[string][]*Blog) error {
	var tagIndex = make(map[string]int)
	for tag, blogs := range tagMap {
		tagIndex[tag] = len(blogs)
	}
	content, err := json.Marshal(tagIndex)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(PublicIndexDir, "tags"), content, 0755)
}

// GenTagIndex 为每个标签分别生成索引。形式是{[博客简要信息列表],标签文章总数}。每10篇博客分一页。
func GenTagIndex(tagMap map[string][]*Blog) {
	for tag, blogs := range tagMap {
		// 先为标签创建一个文件夹。
		tagDir := path.Join(PublicIndexDir, tag)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			log.Println(err)
			continue
		}
		// 然后每10篇博客创建一个索引页。
		total := len(blogs)
		totalPage := (total / 10) + 1
		for i := 1; i <= totalPage; i++ {
			var tagIndex = &TagIndex{
				Total: total,
			}
			if i == totalPage {
				tagIndex.Blogs = blogs[(i-1)*10:]
			} else {
				tagIndex.Blogs = blogs[(i-1)*10 : i*10]
			}
			text, err := json.Marshal(tagIndex)
			if err != nil {
				log.Println(err)
				continue
			}
			if err := ioutil.WriteFile(path.Join(tagDir, fmt.Sprint(i)), text, 0755); err != nil {
				log.Println(err)
			}
		}
	}
}

var metaPattern, _ = regexp.Compile("(?s)^```lw-blog-meta(.*?)```")
var blogs []*Blog

type Blog struct {
	Title string   `json:"title"`
	Date  string   `json:"date"`
	Brev  string   `json:"brev"`
	Tags  []string `json:"tags"`
	Path  string   `json:"path"`
}

func QuickSortBlog(li []*Blog) {
	quickSortBlog(li, 0, len(li)-1)
}

func quickSortBlog(li []*Blog, lo, hi int) {
	if hi-lo < 5 {
		quickSortBlogSelect(li, lo, hi)
		return
	}
	mid := quickSortBlogPartition(li, lo, hi)
	quickSortBlog(li, lo, mid-1)
	quickSortBlog(li, mid+1, hi)
}

func quickSortBlogPartition(li []*Blog, lo, hi int) (mid int) {
	l, r := lo, hi
	midValue := li[lo].Date
	for l < r {
		for l <= hi {
			if li[l].Date < midValue { // 比较处
				break
			}
			l++
		}
		for r >= lo {
			if li[r].Date >= midValue { // 比较处
				break
			}
			r--
		}
		if l < r {
			li[l], li[r] = li[r], li[l]
		} else {
			break
		}
	}
	li[lo], li[r] = li[r], li[lo]
	return r
}

func quickSortBlogSelect(li []*Blog, lo, hi int) {
	var min int
	for ; lo < hi; lo++ {
		min = lo
		for i := lo + 1; i <= hi; i++ {
			if li[i].Date > li[min].Date { // 比较处
				min = i
			}
		}
		if lo != min {
			li[lo], li[min] = li[min], li[lo]
		}
	}
}

type TagIndex struct {
	Blogs []*Blog `json:"blogs"`
	Total int     `json:"total"`
}
