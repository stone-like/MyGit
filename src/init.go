package src

import (
	"errors"
	"fmt"
	"io"
	data "mygit/src/database"
	"os"
	"path/filepath"
)

var ErrorFileAlreadyExists = errors.New("file already exists")

var DEFAULT_BRANCH = "master"

func createInitPath(args []string) string {
	if len(args) == 0 {
		path, _ := os.Getwd()
		return path
	} else {
		return args[0]
	}
}

func gitInit(gitPath string, w io.Writer) error {
	for _, dir := range []string{"objects", "refs/heads"} {
		path := filepath.Join(gitPath, dir)

		if _, err := os.Stat(path); err == nil {
			return ErrorFileAlreadyExists
		}

		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}

	}

	r := &data.Refs{
		Path: gitPath,
	}
	defaultBranchPath := filepath.Join("refs", "heads", DEFAULT_BRANCH)
	//refs/heads/masterを作ろう...

	absPath := filepath.Join(gitPath, defaultBranchPath)
	stat, _ := os.Stat(absPath)
	if stat == nil {
		f, err := os.Create(absPath)
		defer f.Close()
		if err != nil {
			return err
		}
	}
	r.UpdateHead(fmt.Sprintf("ref: %s", defaultBranchPath))

	w.Write([]byte(fmt.Sprintf("Initialized empty Mygit repository in %s\n", gitPath)))

	return nil
}

func StartInit(args []string, w io.Writer) error {
	rootPath, err := filepath.Abs(createInitPath(args))

	if err != nil {
		return err
	}

	//存在しないpathでもエラーは出ないので、ここでエラーを出している
	if _, err := os.Stat(rootPath); err != nil {
		return err
	}

	gitPath := filepath.Join(rootPath, ".git")

	err = gitInit(gitPath, w)

	if err != nil {
		return err
	}

	return nil

}
