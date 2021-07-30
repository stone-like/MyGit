package src

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	con "mygit/src/database/content"
	er "mygit/src/errors"
	"path/filepath"
)

type LogOption struct {
	IsAbbrev bool
	Format   string
	Patch    bool
}

//optionのdecorationは後で実装,display patchも後で

func StartLog(rootPath string, args []string, option *LogOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")

	repo := GenerateRepository(rootPath, gitPath, dbPath)

	revList, err := GenerateRevList(repo, args)
	if err != nil {
		return err
	}

	//ここでshowFnを作っている理由はLogとRevListを分けたいから
	//LogはShowとかの表示用でOptionの情報とかほしい
	//RevListはCommitの順番のQueueを計算用で余計なOptionとかWriterとかの情報はいらない
	showFn := func(c *con.CommitFromMem) error {
		err := ShowCommit(revList, c, option, repo, w)
		if err != nil {
			return err
		}

		return nil
	}
	err = revList.EachCommit(showFn)

	if err != nil {
		return err
	}

	return nil
}

func AbbrObjId(objId string, repo *Repository, option *LogOption) string {
	if option.IsAbbrev {
		return ShortOid(objId, repo.d)
	} else {
		return objId
	}
}

func ShowCommitMedium(c *con.CommitFromMem, option *LogOption, repo *Repository, w io.Writer) error {
	w.Write([]byte(fmt.Sprintf("commit %s\n", AbbrObjId(c.ObjId, repo, option))))
	w.Write([]byte(fmt.Sprintf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)))
	w.Write([]byte(fmt.Sprintf("Date: %s\n", c.Author.ReadableTime())))
	w.Write([]byte("\n"))

	buf := bytes.NewBuffer([]byte(c.Message))

	s := bufio.NewScanner(buf)

	for s.Scan() {
		w.Write([]byte(fmt.Sprintf("     %s\n", s.Text())))
	}

	return nil
}

func ShowCommitOneLine(c *con.CommitFromMem, option *LogOption, repo *Repository, w io.Writer) error {
	w.Write([]byte(fmt.Sprintf("%s %s\n", AbbrObjId(c.ObjId, repo, option), c.GetFirstLineMessage())))

	return nil
}

func ShowPatch(revList *RevList, c *con.CommitFromMem, option *LogOption, repo *Repository, w io.Writer) error {
	if !option.Patch {
		return &er.InvalidFormatError{
			FormatName: option.Format,
		}
	}

	w.Write([]byte("\n"))
	err := PrintCommitDiff(c.FirstParent(), c.ObjId, repo, revList, w)
	if err != nil {
		return err
	}

	return nil
}

func ShowCommit(revList *RevList, c *con.CommitFromMem, option *LogOption, repo *Repository, w io.Writer) error {

	switch option.Format {
	case "":
		err := ShowCommitMedium(c, option, repo, w)
		if err != nil {
			return err
		}
	case "oneline":
		err := ShowCommitOneLine(c, option, repo, w)
		if err != nil {
			return err
		}
	}

	if option.Patch {
		return ShowPatch(revList, c, option, repo, w)
	}

	return nil

}
