package libs

import (
	"context"
	"errors"
	"github.com/saodd/alog"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const ProjectDirname = "tech-blog-content"

// CheckWorkDir 检查当前工作目录是否是项目根目录，不是的话就退出
func CheckWorkDir(c context.Context) error {
	workDir, err := os.Getwd()
	if err != nil {
		alog.CE(c, err)
		return err
	}
	if filepath.Base(workDir) != ProjectDirname {
		err := errors.New("当前路径不是项目根目录(%s)！\n")
		alog.CE(c, err, alog.V{"Dir": ProjectDirname})
		return err
	}
	return nil
}

// RecurListMds 将递归遍历指定目录，返回所有 .md 文件的路径。
func RecurListMds(c context.Context, folder string) (mds []string, err error) {
	files, _ := ioutil.ReadDir(folder)
	for _, file := range files {
		if file.IsDir() {
			// 仅允许文件夹名为数字（年份），且排除2019年
			if year, err := strconv.Atoi(file.Name()); err != nil {
				continue
			} else if year < 2020 {
				continue
			}
			subFolder := filepath.Join(folder, file.Name())
			subMds, err := RecurListMds(c, subFolder)
			if err != nil {
				alog.CE(c, err, alog.V{"Dir": folder})
				return nil, err
			}
			mds = append(mds, subMds...)
		} else {
			if name := file.Name(); len(name) > 3 && name[len(name)-3:] == ".md" {
				mds = append(mds, filepath.ToSlash(filepath.Join(folder, file.Name())))
			}
		}
	}
	return
}
