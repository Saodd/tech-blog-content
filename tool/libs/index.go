package libs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

const PublicIndexDir = "./public/index"

func GenIndexes(blogs []*Blog) error {
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
		return err
	}
	if err := genRootIndex(tagMap); err != nil {
		return err
	}
	if err := genTagIndex(tagMap); err != nil {
		return err
	}
	return nil
}

// genRootIndex 生成一个总索引。形式是 [标签名:文章数]，用于左侧导航栏。
func genRootIndex(tagMap map[string][]*Blog) error {
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

// genTagIndex 为每个标签分别生成索引。形式是{[博客简要信息列表],标签文章总数}。每10篇博客分一页。
func genTagIndex(tagMap map[string][]*Blog) error {
	for tag, blogs := range tagMap {
		// 先为标签创建一个文件夹。
		tagDir := path.Join(PublicIndexDir, tag)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			return err
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
				return err
			}
			if err := ioutil.WriteFile(path.Join(tagDir, strconv.Itoa(i)), text, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}

type TagIndex struct {
	Blogs []*Blog `json:"blogs"`
	Total int     `json:"total"`
}
