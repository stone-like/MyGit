package main

import "mygit/cmd"

func main() {
	cmd.Execute()

	// var str string

	// var testb []byte
	// root := "9413"

	// rootLen := len(root)

	// for i := 0; i < rootLen; i += 2 {
	// 	s1 := string(root[i])
	// 	s2 := string(root[i+1])
	// 	i1, err := strconv.ParseInt(s1, 16, 0)

	// 	if err != nil {
	// 		panic("err")
	// 	}

	// 	i2, err := strconv.ParseInt(s2, 16, 0)
	// 	if err != nil {
	// 		panic("err")
	// 	}

	// 	bin1 := fmt.Sprintf("%04b", i1)
	// 	bin2 := fmt.Sprintf("%04b", i2)

	// 	res, err := strconv.ParseInt(bin1+bin2, 2, 0)
	// 	if err != nil {
	// 		panic("err")
	// 	}

	// 	testb = append(testb,byte(res))
	// 	fmt.Println(strconv.Itoa(int(res)))
	// 	str += strconv.Itoa(int(res))
	// }

	// fmt.Println(testb)

	// curDir, _ := os.Getwd()

	// tempDir1, _ := ioutil.TempDir(curDir, "tempDir")
	// f1, _ := ioutil.TempFile(tempDir1, "test")

	// defer f1.Close()

	// _, _ = f1.Write([]byte(string(testb)))

}
