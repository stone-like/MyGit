package src

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrorFileAlreadyExists = errors.New("file already exists")

func createInitPath(args []string) string {
	if len(args) == 0 {
		path, _ := os.Getwd()
		return path
	} else {
		return args[0]
	}
}

func gitInit(gitPath string) error {
	for _, dir := range []string{"objects", "refs"} {
		path := filepath.Join(gitPath, dir)

		if _, err := os.Stat(path); err == nil {
			return ErrorFileAlreadyExists
		}

		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}

	}

	return nil
}

func StartInit(args []string) error {
	rootPath, err := filepath.Abs(createInitPath(args))

	if err != nil {
		return err
	}

	//存在しないpathでもエラーは出ないので、ここでエラーを出している
	if _, err := os.Stat(rootPath); err != nil {
		return err
	}

	gitPath := filepath.Join(rootPath, ".git")

	err = gitInit(gitPath)

	if err != nil {
		return err
	}

	return nil

}
