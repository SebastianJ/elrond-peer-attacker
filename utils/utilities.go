package utils

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RandomElementFromArray fetches a random element from an array
func RandomElementFromArray(items []string) string {
	randomIndex := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(len(items))
	item := items[randomIndex]

	return item
}

// RandomElementFromIntSlice fetches a random element from an array
func RandomElementFromIntSlice(items []int) int {
	randomIndex := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(len(items))
	item := items[randomIndex]

	return item
}

// RandomElementFromUint32Slice fetches a random element from an array
func RandomElementFromUint32Slice(items []uint32) uint32 {
	randomIndex := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(len(items))
	item := items[randomIndex]

	return item
}

// ArrayFromFile - fetch a list of ip addresses from a specified file
func ArrayFromFile(filePath string) (lines []string, err error) {
	data, err := ReadFileToString(filePath)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		lines = strings.Split(string(data), "\n")
		// Remove extra line introduced by strings.Split - see https://play.golang.org/p/sNsAc2xVDT
		if strings.Contains(data, "\n") {
			lines = lines[:len(lines)-1]
		}
	}

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
		return "", fmt.Errorf("file %s doesn't exist", filePath)
	}
}
