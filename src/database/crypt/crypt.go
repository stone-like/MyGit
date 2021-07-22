package crypt

import (
	"fmt"
	"strconv"
)

func CreateH40(objId string) (string, error) {
	var bs []byte
	root := objId
	rootLen := len(root)

	for i := 0; i < rootLen; i += 2 {

		s1 := string(root[i])
		s2 := string(root[i+1])
		i1, err := strconv.ParseInt(s1, 16, 64)
		if err != nil {
			return "", err
		}

		i2, err := strconv.ParseInt(s2, 16, 64)
		if err != nil {
			return "", err
		}

		bin1 := fmt.Sprintf("%04b", i1)
		bin2 := fmt.Sprintf("%04b", i2)

		res, err := strconv.ParseInt(bin1+bin2, 2, 64)
		if err != nil {
			return "", err
		}
		bs = append(bs, byte(res))
	}

	return string(bs), nil

}
