package utils

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RandomElementFromArray fetches a random element from an array
func RandomElementFromArray(items []string) string {
	rand.Seed(time.Now().Unix())
	item := items[rand.Intn(len(items))]

	return item
}

// ArrayFromFile - fetch a list of ip addresses from a specified file
func ArrayFromFile(filePath string) ([]string, error) {
	data, err := ReadFileToString(filePath)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	return lines, nil
}

func globFiles(pattern string) ([]string, error) {
	files, err := filepath.Glob(pattern)

	if err != nil {
		return nil, err
	}

	return files, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// ReadFileToString - check if a file exists, proceed to read it to memory if it does
func ReadFileToString(filePath string) (string, error) {
	if fileExists(filePath) {
		data, err := ioutil.ReadFile(filePath)

		if err != nil {
			return "", err
		}

		return string(data), nil
	} else {
		return "", nil
	}
}
