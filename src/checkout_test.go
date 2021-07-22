package src

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Index_Updated(t *testing.T) {
	path := PrepareCompareTwoCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	rootPath := filepath.Join(curDir, "tempDir")

	var buf bytes.Buffer
	//HEAD->HEAD^に戻す
	err = StartCheckout(rootPath, []string{"@^"}, &buf)
	assert.NoError(t, err)

	diffBuf := new(bytes.Buffer)
	//checkoutでwortkspaceとindexがかわらなくなっているはず
	//いずれuntrackedとかをのこすようになるとtestは増やさなければいけないが、今回用意したprepareではuntrackedだったりuncommitなものはないのでOK
	err = StartStatus(diffBuf, rootPath, true)
	assert.NoError(t, err)

	assert.Equal(t, "nothing to commit, working tree clean", diffBuf.String())
}

//Conflictの定義として、OldCommitとNewCommitから生成されたTreeDiffにあるPathとIndex,Workspaceを比較する、つまりCommitしていないIndexやWorkSpaceがDiffのファイルとコンフリクトしてしまうとうまくcheckoutできない
//対して、TreeDiffにはないファイルなら別にコンフリクトしないので別に良い
//ただTreeDiffの親にuntrackedがあるとだめ
func Test_ConflictUntrackParent(t *testing.T) {
	path := PrepareCompareTwoCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	rootPath := filepath.Join(curDir, "tempDir")

	xxxPath := filepath.Join(rootPath, "xxx")

	//xxx/addedがUntracked
	CreateFiles(t, xxxPath, "added.txt", "test")

	var buf bytes.Buffer
	//HEAD->HEAD^に戻す
	err = StartCheckout(rootPath, []string{"@^"}, &buf)
	assert.NoError(t, err)

}
