package libs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/saodd/alog"
	"resty.dev/v3"
)

var (
	ServerAddress string
	ServerToken   string
)

var restyClient = resty.New().
	SetTimeout(time.Second * 10)

type Response struct {
	Code int   `json:"code"`
	Time int64 `json:"ts"`
	Data any   `json:"data"`
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
		if ok {
			fmt.Printf("Hash： %s | %s ，文件： %s\n", b.Hash, serverBlog.Hash, b.Path)
		} else {
			fmt.Printf("Hash： %s | _ ，文件： %s\n", b.Hash, b.Path)
		}
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

	var data []*Blog
	var result Response
	result.Data = &data

	resp, err := restyClient.R().
		SetContext(c).
		SetHeader("X-STAFF-TOKEN", ServerToken).
		SetResult(&result).
		Get(u)

	if err != nil {
		alog.CE(c, err, alog.V{"u": u})
		return nil, err
	}
	if resp.StatusCode() != 200 {
		err := errors.New("获取Hash失败：" + resp.Status())
		alog.CE(c, err)
		return nil, err
	}
	if result.Code != 0 {
		err := fmt.Errorf("resp code：%d", result.Code)
		alog.CE(c, err, alog.V{"result": result})
		return nil, err
	}

	return data, nil
}

func UpsertBlog(c context.Context, blog *Blog) error {
	u := ServerAddress + "/upsert_article"

	var result Response
	resp, err := restyClient.R().
		SetContext(c).
		SetHeader("X-STAFF-TOKEN", ServerToken).
		SetBody(blog).
		SetResult(&result).
		Post(u)

	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New("POST推送数据失败：" + resp.Status())
	}
	if result.Code != 0 {
		return fmt.Errorf("resp code：%d", result.Code)
	}
	log.Println("推送博客：", blog.Path)

	return nil
}

func DeleteBlog(c context.Context, blog *Blog) error {
	u := ServerAddress + "/delete_article"

	var result Response
	resp, err := restyClient.R().
		SetContext(c).
		SetHeader("X-STAFF-TOKEN", ServerToken).
		SetBody(map[string]string{"path": blog.Path}).
		SetResult(&result).
		Post(u)

	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New("POST推送数据失败：" + resp.Status())
	}
	if result.Code != 0 {
		return fmt.Errorf("resp code：%d", result.Code)
	}

	return nil
}

func init() {
	if mode := os.Getenv("RUN_MODE"); mode == "gh-actions" {
		ServerAddress = "https://lewinblog.com/api/blog/staff"
	} else {
		ServerAddress = "http://localhost:20001/blog/staff"
	}
	ServerToken = os.Getenv("JULIET_POST_TOKEN")
	if ServerToken == "" {
		log.Fatalln("未设置环境变量 JULIET_POST_TOKEN")
	}
}
