package src

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

//CASが複数ある状態でBCAを見つけられることをTest

//TestDataはGで一回masterにtopicをマージしている
// A <- B <- C <- G <- H <- I [master]
//       \       /
//         D <- E <- F [topic]
//ここでBCAがBではなくEであってほしい

//DにparentOne,Two,Staleが引き継がれてしまうがいいのか？<-ok

func TestCas(t *testing.T) {

	cur, err := os.Getwd()
	assert.NoError(t, err)

	testPath := filepath.Join(cur, "testData/Cas")
	gitPath := filepath.Join(testPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(testPath, gitPath, dbPath)

	headObjId, err := repo.r.ReadHead()
	assert.NoError(t, err)

	rev, err := ParseRev("topic")
	assert.NoError(t, err)
	mergeObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	//baseObjIdがEであってほしい
	baseObjId, err := GetBCA(headObjId, mergeObjId, repo.d)
	assert.NoError(t, err)

	assert.Equal(t, "b914e5dca3d334f3f69728a6b515cefa069fff07", baseObjId)

}

//priorityQueueの性質上Dの後にBが来るのでBは絶対staleとなりresultにこないはずだが...
// A <- B <- C  <----  J <- K [master]
//       \            /
//         D <- E <- F [topic2]
//          \
//            G <- H [topic1]

//crossMergeの時BCAが一個以上あるらしいが,crossMergeの時はBCA同時に共通祖先がないのでCAS探しを多重にするという戦略は取れないはず
//なのでcrossMerge以外でBCAが一個以上あるケースを探したいが...見つからないのでいったん保留
//todo -> BCAが二つ以上存在するケースがcrossMergeしかわからずfilter_commitの存在理由がわからない
func TestCasComplex(t *testing.T) {

	cur, err := os.Getwd()
	assert.NoError(t, err)

	testPath := filepath.Join(cur, "testData/CasComplex")
	gitPath := filepath.Join(testPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(testPath, gitPath, dbPath)

	headObjId, err := repo.r.ReadHead()
	assert.NoError(t, err)

	rev, err := ParseRev("topic")
	assert.NoError(t, err)
	mergeObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	//baseObjIdがDであってほしい
	baseObjId, err := GetBCA(headObjId, mergeObjId, repo.d)
	assert.NoError(t, err)

	assert.Equal(t, "a7840d24470a454e8520754dbdb4cfa4705f163b", baseObjId)

}
