package libs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/saodd/alog"
)

var (
	ServerAddress string
	ServerToken   string
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 10,
	},
	Timeout: time.Second * 10,
}

func SyncServer(c context.Context, blogs []*Blog) error {
	toPut, toDel, err := CheckServerBlogs(c, blogs)
	if err != nil {
		alog.CE(c, err)
		return err
	}
	for _, blog := range toPut {
		if err := UpsertBlog(c, blog); err != nil {
			alog.CE(c, err)
			return err
		}
	}
	for _, blog := range toDel {
		if err := DeleteBlog(c, blog); err != nil {
			alog.CE(c, err)
			return err
		}
	}
	return nil
}

func CheckServerBlogs(c context.Context, blogs []*Blog) (toPut, toDel []*Blog, err error) {
	serverBlogs, err := GetServerBlogs(c)
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

func GetServerBlogs(c context.Context) ([]*Blog, error) {
	u := ServerAddress + "/list_hash"
	req, _ := http.NewRequestWithContext(c, "GET", u, nil)
	req.Header.Set("X-STAFF-TOKEN", ServerToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		alog.CE(c, err, alog.V{"u": u})
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err := errors.New("获取Hash失败：" + resp.Status)
		alog.CE(c, err)
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		alog.CE(c, err)
		return nil, err
	}
	var body = struct {
		Data []*Blog `json:"data"`
	}{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		alog.CE(c, err, alog.V{"resp": string(bodyBytes)})
		return nil, err
	}
	return body.Data, nil
}

func UpsertBlog(c context.Context, blog *Blog) error {
	u := ServerAddress + "/upsert_article"
	reqBody, _ := json.Marshal(blog)
	req, _ := http.NewRequestWithContext(c, "POST", u, bytes.NewReader(reqBody))
	req.Header.Set("X-STAFF-TOKEN", ServerToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("POST推送数据失败：" + resp.Status)
	}
	log.Println("推送博客：", blog.Path)

	return nil
}

func DeleteBlog(c context.Context, blog *Blog) error {
	u := ServerAddress + "/delete_article"
	reqBody, _ := json.Marshal(map[string]string{"path": blog.Path})
	req, _ := http.NewRequestWithContext(c, "POST", u, bytes.NewReader(reqBody))
	req.Header.Set("X-STAFF-TOKEN", ServerToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("POST推送数据失败：" + resp.Status)
	}
	log.Println("删除博客：", blog.Path)

	return nil
}

func init() {
	if mode := os.Getenv("RUN_MODE"); mode == "gh-actions" {
		ServerAddress = "https://lewinblog.com/api/blog/staff"
	} else {
		ServerAddress = "http://localhost:20001/blog/staff"
	}
	ServerToken = os.Getenv("JULIET_POST_TOKEN")
}
