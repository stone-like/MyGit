package src

// func Test_Diff3(t *testing.T) {
// 	cur, err := os.Getwd()
// 	assert.NoError(t, err)
// 	testDataPath := filepath.Join(cur, "testdata/diff3")
// 	originPath := filepath.Join(testDataPath, "origin.txt")
// 	aPath := filepath.Join(testDataPath, "a.txt")
// 	bPath := filepath.Join(testDataPath, "b.txt")
// 	originContent, err := ioutil.ReadFile(originPath)
// 	assert.NoError(t, err)
// 	aContent, err := ioutil.ReadFile(aPath)
// 	assert.NoError(t, err)
// 	bContent, err := ioutil.ReadFile(bPath)
// 	assert.NoError(t, err)
// 	originAEdits := myers.ComputeEdits(span.URIFromPath("a.txt"), string(aContent), string(originContent))
// 	originBEdits := myers.ComputeEdits(span.URIFromPath("b.txt"), string(bContent), string(originContent))

// 	fmt.Println(originAEdits, originBEdits)
// }
