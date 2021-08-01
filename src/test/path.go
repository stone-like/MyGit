package test

import "os"

func GenerateRamdomTestPath() string {
	return os.TempDir()
}
