package src

import (
	e "mygit/src/errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func Test_ParseRev(t *testing.T) {

	for _, d := range []struct {
		title        string
		targetString string
		expectedObj  BranchObj
	}{
		{"parseParent", `master^`, &Parent{
			Rev: &Ref{
				Name: "master",
			},
			ParentNum: 1,
		}},
		{"parseNestedParent", `master^^`, &Parent{
			Rev: &Parent{
				Rev: &Ref{
					Name: "master",
				},
				ParentNum: 1,
			},
			ParentNum: 1,
		}},
		{"parseAncestor", `master~42`, &Ancestor{
			Rev: &Ref{
				Name: "master",
			},
			N: 42,
		}},
		{"parseNestedAncestor", `master~42^`, &Parent{
			Rev: &Ancestor{
				Rev: &Ref{
					Name: "master",
				},
				N: 42,
			},
			ParentNum: 1,
		},
		},
		{"parseAlias", `@^`, &Parent{
			Rev: &Ref{
				Name: "HEAD",
			},
			ParentNum: 1,
		}},
	} {
		t.Run(d.title, func(t *testing.T) {
			ret, err := ParseRev(d.targetString)
			assert.NoError(t, err)

			if diff := cmp.Diff(d.expectedObj, ret); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}

//コミットメッセージが複雑なやつを作る
// func Test_CreateCommitMessage(t *testing.T) {
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	tempPath := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	xxxPath := filepath.Join(tempPath, "xxx")
// 	err = os.MkdirAll(xxxPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	helloName := CreateFiles(t, tempPath, "hello.txt", "test\n")
// 	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

// 	rel1, err := filepath.Rel(tempPath, helloName)
// 	assert.NoError(t, err)
// 	rel2, err := filepath.Rel(tempPath, dummyName)
// 	assert.NoError(t, err)
// 	is := []string{tempPath}
// 	var buf bytes.Buffer
// 	err = StartInit(is, &buf)
// 	assert.NoError(t, err)
// 	ss := []string{rel1, rel2}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "this is commitMessage\n\nhardToRead")
// 	assert.NoError(t, err)

// 	// return func() {
// 	// 	os.RemoveAll(tempPath)
// 	// }
// }

//コミットが３つあるやつを作る
// func Test_CreateThreeCommit(t *testing.T) {
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	tempPath := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	xxxPath := filepath.Join(tempPath, "xxx")
// 	err = os.MkdirAll(xxxPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	helloName := CreateFiles(t, tempPath, "hello.txt", "test\n")
// 	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

// 	rel1, err := filepath.Rel(tempPath, helloName)
// 	assert.NoError(t, err)
// 	rel2, err := filepath.Rel(tempPath, dummyName)
// 	assert.NoError(t, err)
// 	is := []string{tempPath}
// 	var buf bytes.Buffer
// 	err = StartInit(is, &buf)
// 	assert.NoError(t, err)
// 	ss := []string{rel1, rel2}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
// 	assert.NoError(t, err)

// 	CreateFiles(t, xxxPath, "dummy2.txt", "test2\n")
// 	ss = []string{"."}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
// 	assert.NoError(t, err)

// 	CreateFiles(t, xxxPath, "dummy3.txt", "test2\n")
// 	ss = []string{"."}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
// 	assert.NoError(t, err)

// 	// return func() {
// 	// 	os.RemoveAll(tempPath)
// 	// }
// }

func Test_ResolveRev(t *testing.T) {

	for _, d := range []struct {
		title         string
		targetString  string
		expectedObjId string
	}{
		{"resolveDobuleParent", `@^^`, "c130d5a8ca05710c56eab9ff35e5e52d1180fc72"},
		{"resolveOneParentOneAncestor", `@^~1`, "c130d5a8ca05710c56eab9ff35e5e52d1180fc72"},
		{"resolveOneAncestorOneParent", `@~1^`, "c130d5a8ca05710c56eab9ff35e5e52d1180fc72"},
		{"resolveDoubhleAncestor", `@~2`, "c130d5a8ca05710c56eab9ff35e5e52d1180fc72"},
	} {
		t.Run(d.title, func(t *testing.T) {
			ret, err := ParseRev(d.targetString)
			assert.NoError(t, err)

			curDir, err := os.Getwd()
			assert.NoError(t, err)
			rootPath := filepath.Join(curDir, "testData/threeCommitData")
			gitPath := filepath.Join(rootPath, ".git")
			dbPath := filepath.Join(gitPath, "objects")
			repo := GenerateRepository(rootPath, gitPath, dbPath)
			objId, err := ResolveRev(ret, repo)
			assert.NoError(t, err)
			assert.Equal(t, d.expectedObjId, objId)
		})

	}

}

//Dirの名前がかぶっているやつを作る
// func Test_Dir(t *testing.T) {
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	tempPath := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	xxxPath := filepath.Join(tempPath, "xxx")
// 	err = os.MkdirAll(xxxPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	CreateFiles(t, tempPath, "hello.txt", "test\n")
// 	CreateFiles(t, tempPath, "hello1.txt", "test1\n")
// 	CreateFiles(t, tempPath, "hello2.txt", "test2\n")
// 	CreateFiles(t, tempPath, "hello3.txt", "test3\n")
// 	CreateFiles(t, tempPath, "hello4.txt", "test4\n")
// 	CreateFiles(t, tempPath, "hello5.txt", "test5\n")
// 	CreateFiles(t, tempPath, "hello6.txt", "test6\n")
// 	CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

// 	is := []string{tempPath}
// 	var buf bytes.Buffer
// 	err = StartInit(is, &buf)
// 	assert.NoError(t, err)
// 	ss := []string{"."}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "this is commitMessage\n\nhardToRead")
// 	assert.NoError(t, err)

// 	// return func() {
// 	// 	os.RemoveAll(tempPath)
// 	// }
// }

func TestGetError(t *testing.T) {

	for _, d := range []struct {
		title         string
		targetString  string
		expectedError error
	}{
		{"AmbiguousError", `18`, &e.InvalidObjectError{
			Message: "short SHA1 18 is ambiguos",
			Hint: []string{
				"The candidates are:",
				" 1800a6 commit 2021-07-18 - this is commitMessage",
				" 180cf8 blob",
			},
			CriticalInfo: "Not a valid object name: 18",
		}},
		{"ConversionError", `d234c5e057fe32c676ea67e8cb38f4625ddaeb54`, &e.ObjConvertionError{
			Type:         "commit",
			Message:      "object d234c5e057fe32c676ea67e8cb38f4625ddaeb54 is a blob, not a commit",
			CriticalInfo: "Not a valid object name: d234c5e057fe32c676ea67e8cb38f4625ddaeb54",
		}},
		{"ConversionError", `d234`, &e.ObjConvertionError{
			Type:         "commit",
			Message:      "object d234c5e057fe32c676ea67e8cb38f4625ddaeb54 is a blob, not a commit",
			CriticalInfo: "Not a valid object name: d234",
		}},
		{"NoError", "1800a66d7a53e2987826b5b13b7239458a71b4bc", nil},
	} {
		t.Run(d.title, func(t *testing.T) {
			ret, err := ParseRev(d.targetString)
			assert.NoError(t, err)

			curDir, err := os.Getwd()
			assert.NoError(t, err)
			rootPath := filepath.Join(curDir, "testData/ambDir")
			gitPath := filepath.Join(rootPath, ".git")
			dbPath := filepath.Join(gitPath, "objects")
			repo := GenerateRepository(rootPath, gitPath, dbPath)
			_, err = ResolveRev(ret, repo)
			if diff := cmp.Diff(d.expectedError, err); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})

	}

}

func TestMultipleParent(t *testing.T) {
	fn := PrepareMergedFileTreeSame(t)
	t.Cleanup(fn)

	for _, d := range []struct {
		title         string
		targetString  string
		expectedObjId string
	}{
		{"commitB", `@^`, "6c6ec1f7ed45ecb6002e7efe5bfb277ead04d7e1"},
		{"commitC", `@^2`, "a22fd54a34895183dc1069be64e169761c9c9a99"},
	} {
		t.Run(d.title, func(t *testing.T) {
			ret, err := ParseRev(d.targetString)
			assert.NoError(t, err)

			curDir, err := os.Getwd()
			assert.NoError(t, err)
			rootPath := filepath.Join(curDir, "testData/multipleParentRev")
			gitPath := filepath.Join(rootPath, ".git")
			dbPath := filepath.Join(gitPath, "objects")
			repo := GenerateRepository(rootPath, gitPath, dbPath)
			objId, err := ResolveRev(ret, repo)
			assert.NoError(t, err)
			assert.Equal(t, d.expectedObjId, objId)
		})

	}
}
