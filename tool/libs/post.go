package libs

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

var (
	ServerAddress string
	ServerToken   string
)

func PostPathsToServer(blogs []*Blog) error {
	var paths = make([]string, len(blogs))
	for i, blog := range blogs {
		paths[i] = blog.Path
	}
	reqBody, _ := json.Marshal(paths)
	req, _ := http.NewRequest("POST", ServerAddress, bytes.NewReader(reqBody))
	req.Header.Set("X-Github-PostToken", ServerToken)
	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		return errors.New("推送数据失败：" + resp.Status)
	}
	return err
}

func init() {
	if mode := os.Getenv("RUN_MODE"); mode == "" {
		ServerAddress = "http://localhost/blog/paths"
	} else {
		ServerAddress = "https://api.lewinblog.com/blog/paths"
	}
	ServerToken = os.Getenv("JULIET_POST_TOKEN")
}
