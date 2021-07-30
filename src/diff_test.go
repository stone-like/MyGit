	"mygit/src/crypt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"github.com/google/go-cmp/cmp"
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

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")

	beforeBlob, err := CreateObjIdFromPath(helloPath)
	f1, _ := os.OpenFile(helloPath, os.O_RDWR|os.O_CREATE, os.ModePerm)

	afterBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\nindex %s..%s 100644\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n+change1\n", ShortOid(beforeBlob, repo.d), ShortOid(afterBlob, repo.d))

	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	err := os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)

	err := os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	beforeBlob, err := CreateObjIdFromPath(helloPath)

	f1, _ := os.OpenFile(helloPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	afterBlob, err := CreateObjIdFromPath(helloPath)
	assert.NoError(t, err)

	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\nindex %s..%s\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n+change1\n", ShortOid(beforeBlob, repo.d), ShortOid(afterBlob, repo.d))
	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	beforeBlob, err := CreateObjIdFromPath(helloPath)
	err = os.RemoveAll(helloPath)
	expected := fmt.Sprintf("diff --git a/hello.txt b/hello.txt\ndeleted file mode 100644\nindex %s..000000\n--- a/hello.txt\n+++ b/hello.txt\n@@ -1 +1 @@\n-test\n", ShortOid(beforeBlob, repo.d))
	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
//diffの順番が変わってしまう..DiffHeadIndex
//s.IndexChangesの順番が変動するのが問題っぽい,ここを治す
	helloPath := filepath.Join(curDir, "tempDir/hello.txt")
	err := os.Chmod(helloPath, 0777)
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
	err := os.RemoveAll(filepath.Join(tempPath, "hello.txt"))
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
	err := StartAdd(tempPath, "test", "test@example.com", "test", ss)
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