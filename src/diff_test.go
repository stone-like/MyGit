package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mygit/src/crypt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// //diffの原理は大体理解したのでいったん既存の奴を使う、ブランチまで終わったら自力で実装
// func TestDiff(t *testing.T) {
// 	cur, err := os.Getwd()
// 	assert.NoError(t, err)
// 	testDataPath := filepath.Join(cur, "testdata")
// 	prevPath := filepath.Join(testDataPath, "prev.txt")
// 	diffPath := filepath.Join(testDataPath, "diff.txt")
// 	s1, err := ioutil.ReadFile(prevPath)
// 	assert.NoError(t, err)
// 	s2, err := ioutil.ReadFile(diffPath)
// 	assert.NoError(t, err)
// 	edits := myers.ComputeEdits(span.URIFromPath("a.txt"), string(s1), string(s2))
// 	diff := fmt.Sprint(gotextdiff.ToUnified("a.txt", "b.txt", string(s1), edits))

// 	fmt.Print(diff)
// }

var curDir, _ = os.Getwd()
var tempPath = filepath.Join(curDir, "tempDir")
var gitPath = filepath.Join(tempPath, ".git")
var dbPath = filepath.Join(gitPath, "objects")
var repo = GenerateRepository(tempPath, gitPath, dbPath)

func ReadFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func CreateObjIdFromPath(path string) (string, error) {
	content, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	blob := &con.Blob{
		Content: content,
	}
	headerCon := data.GetStoreHeaderContent(blob)
	objId := crypt.HexDigestBySha1(headerCon)

	return objId, nil
}
func CreateObjIdFromContent(c string) (string, error) {

	blob := &con.Blob{
		Content: c,
	}
	headerCon := data.GetStoreHeaderContent(blob)
	objId := crypt.HexDigestBySha1(headerCon)

	return objId, nil
}

func TestDiffStatusModContent_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")

	beforeBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)
	f1, _ := os.OpenFile(helloPath, os.O_RDWR|os.O_CREATE, os.ModePerm)

	defer f1.Close()
	f1.Write([]byte("change1\n"))

	afterBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{})
	assert.NoError(t, err)

	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\nindex %s..%s 100644\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n+change1\n", ShortOid(beforeBlob, repo.d), ShortOid(afterBlob, repo.d))

	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestDiffStatusModMode_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	err := os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{})
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusModModeAndContent_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	err := os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	beforeBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	f1, _ := os.OpenFile(helloPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1\n"))

	afterBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{})
	assert.NoError(t, err)

	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\nindex %s..%s\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n+change1\n", ShortOid(beforeBlob, repo.d), ShortOid(afterBlob, repo.d))

	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestDiffStatusDeleted_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	beforeBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	err = os.RemoveAll(helloPath)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{})
	assert.NoError(t, err)

	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\ndeleted file mode 100644\nindex %s..000000\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n", ShortOid(beforeBlob, repo.d))

	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

//diffの順番が変わってしまう..DiffHeadIndex
//s.IndexChangesの順番が変動するのが問題っぽい,ここを治す
func TestDiffStatusModModeAndContent_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	err := os.Chmod(helloPath, 0777)
	assert.NoError(t, err)
	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/xxx/dummy2.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("require multipleline\nfunc changed()\nchanged executed\n\n\n\n\n\n\n\n\nSimulate End\nSimulate Restart\n"))

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{Cached: true})
	assert.NoError(t, err)

	str := buf.String()
	expected := `diff --git a/hello.txt b/hello.txt
old mode 100644
new mode 100755
diff --git a/xxx/dummy2.txt b/xxx/dummy2.txt
index e738bb..8652b4 100644
--- a/xxx/dummy2.txt
+++ b/xxx/dummy2.txt
@@ -1,5 +1,6 @@
 require multipleline
-func simulate()
+func changed()
+changed executed
 
 
 
@@ -9,3 +10,4 @@
 
 
 Simulate End
+Simulate Restart
`
	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestDiffStatusDeleted_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	err := os.RemoveAll(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)

	//git rmを実装するまではindexを削除しなければindexからdeleteできない
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{Cached: true})
	assert.NoError(t, err)

	str := buf.String()

	expected := `diff --git a/hello.txt b/hello.txt
deleted file mode 100644
index 9daeaf..000000
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
-test
`

	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestDiffStatusAdded_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	CreateFiles(t, tempPath, "added.txt", "added\n")

	ss := []string{"."}
	err := StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, &DiffOption{Cached: true})
	assert.NoError(t, err)

	str := buf.String()
	expected := `diff --git a/added.txt b/added.txt
new file mode 100644
index 000000..d5f7fc
--- a/added.txt
+++ b/added.txt
@@ -1 +1 @@
+added
`

	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func PrepareConflictMerge(t *testing.T) string {

	// A -> B -> D master
	//   \    /
	//     C   test1

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "initial\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("test1Changed\n"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("masterChanged\n"))
	f2.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err = StartMerge(mc, &buf)
	assert.NoError(t, err)

	return tempPath
}

//TODO conflictのときdiff --gitではなく--ccとなったり、stage未選択でのconflictDiff表示は現在未実装
//本家はoursとtheirsでも、元ファイルの-test1Changedとかも表示されるのでやっぱりdiffは自作しなくてはいけなさそう
func TestDiffStatusConflicted(t *testing.T) {
	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	initialObjId, err := CreateObjIdFromContent("initial\n")
	assert.NoError(t, err)
	test1ObjId, err := CreateObjIdFromContent("test1Changed\n")
	assert.NoError(t, err)
	masterObjId, err := CreateObjIdFromContent("masterChanged\n")
	assert.NoError(t, err)

	content, err := ioutil.ReadFile(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	curWorkSpaceObjId, err := CreateObjIdFromContent(string(content))
	assert.NoError(t, err)

	for _, d := range []struct {
		title       string
		targetStage int
		expected    string
	}{
		{
			"base",
			1,
			fmt.Sprintf(
				`* Unmerged path hello.txt
diff --git a/hello.txt b/hello.txt
index %s..%s 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1,7 @@
-initial
+<<<<<<< HEAD
+masterChanged
+
+=======
+test1Changed
+
+>>>>>>> test1
`, repo.d.ShortObjId(initialObjId), repo.d.ShortObjId(curWorkSpaceObjId)),
		},
		{
			"ours",
			2,
			fmt.Sprintf(
				`* Unmerged path hello.txt
diff --git a/hello.txt b/hello.txt
index %s..%s 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1,7 @@
+<<<<<<< HEAD
 masterChanged
+
+=======
+test1Changed
+
+>>>>>>> test1
`, repo.d.ShortObjId(masterObjId), repo.d.ShortObjId(curWorkSpaceObjId)),
		},
		{
			"theirs",
			3,
			fmt.Sprintf(
				`* Unmerged path hello.txt
diff --git a/hello.txt b/hello.txt
index %s..%s 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1,7 @@
+<<<<<<< HEAD
+masterChanged
+
+=======
 test1Changed
+
+>>>>>>> test1
`, repo.d.ShortObjId(test1ObjId), repo.d.ShortObjId(curWorkSpaceObjId)),
		},
	} {
		t.Run(d.title, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := StartDiff(buf, tempPath, &DiffOption{Stage: d.targetStage})
			assert.NoError(t, err)

			str := buf.String()

			if diff := cmp.Diff(d.expected, str); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}
