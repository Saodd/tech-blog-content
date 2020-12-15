package libs

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	ServerAddress string
	ServerToken   string
)

func SyncServer(blogs []*Blog) {
	toPut, toDel, err := CheckServerBlogs(blogs)
	if err != nil {
		log.Fatal(err)
	}
	for _, blog := range toPut {
		if err := PutBlog(blog); err != nil {
			log.Println(err)
		}
	}
	for _, blog := range toDel {
		if err := DeleteBlog(blog); err != nil {
			log.Println(err)
		}
	}
}

func CheckServerBlogs(blogs []*Blog) (toPut, toDel []*Blog, err error) {
	serverBlogs, err := GetServerBlogs()
	if err != nil {
		return nil, nil, err
	}

	var serverBlogMap = map[string]*Blog{}
	for _, b := range serverBlogs {
		serverBlogMap[b.Path] = b
	}

	for _, b := range blogs {
		serverBlog, ok := serverBlogMap[b.Path]
		if !ok || serverBlog.Hash != b.Hash {
			toPut = append(toPut, b)
		}
		if ok {
			delete(serverBlogMap, b.Path)
		}
	}
	for _, v := range serverBlogMap {
		toDel = append(toDel, v)
	}
	return toPut, toDel, nil
}

func GetServerBlogs() ([]*Blog, error) {
	u := ServerAddress + "/all-hash"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("X-Github-PostToken", ServerToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("推送数据失败：" + resp.Status)
	}
	defer resp.Body.Close()

	var serverBlogs []*Blog
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &serverBlogs); err != nil {
		return nil, err
	}
	return serverBlogs, nil
}

func PutBlog(blog *Blog) error {
	u := ServerAddress + "/content"
	reqBody, _ := json.Marshal(blog)
	req, _ := http.NewRequest("PUT", u, bytes.NewReader(reqBody))
	req.Header.Set("X-Github-PostToken", ServerToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("PUT推送数据失败：" + resp.Status)
	}
	defer resp.Body.Close()

	return nil
}

func DeleteBlog(blog *Blog) error {
	u := ServerAddress + "/content/" + blog.Path
	req, _ := http.NewRequest("DELETE", u, nil)
	req.Header.Set("X-Github-PostToken", ServerToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("DELETE推送数据失败：" + resp.Status)
	}
	defer resp.Body.Close()

	return nil
}

func init() {
	if mode := os.Getenv("RUN_MODE"); mode == "gh-actions" {
		ServerAddress = "https://api.lewinblog.com/blog"
	} else {
		ServerAddress = "http://localhost/blog"
	}
	ServerToken = os.Getenv("JULIET_POST_TOKEN")
}
