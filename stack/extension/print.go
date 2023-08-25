package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var (
	extensionName = filepath.Base(os.Args[0]) // extension name has to match the filename
	printPrefix   = fmt.Sprintf("[%s]", extensionName)

	debug = func() bool {
		if v := os.Getenv("SECURE_LAMBDA_URL_DEBUG"); v == "" || v == "true" {
			return true
		}
		return false
	}()
)

// println adds a prefix and prints the given values. It ignores printing if 'debug' disabled
func println(args ...any) {
	if !debug {
		return
	}
	args = append([]any{printPrefix}, args...)
	fmt.Println(args...)
}

// prettyPrint prepares the given value for printing by formatting it to json
func prettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}
