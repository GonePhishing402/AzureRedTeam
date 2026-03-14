package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDir creates the directory (and parents) if it does not exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0750)
}

// WriteTextFile writes text content to path, creating/truncating the file.
func WriteTextFile(path, content string) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}

// WriteBinaryFile writes raw bytes to path.
func WriteBinaryFile(path string, data []byte) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// log helpers — send formatted lines to the GUI channel

func logInfo(ch chan<- string, format string, args ...interface{}) {
	ch <- fmt.Sprintf("[*] "+format, args...)
}

func logOK(ch chan<- string, format string, args ...interface{}) {
	ch <- fmt.Sprintf("[+] "+format, args...)
}

func logWarn(ch chan<- string, format string, args ...interface{}) {
	ch <- fmt.Sprintf("[!] "+format, args...)
}

func logFail(ch chan<- string, format string, args ...interface{}) {
	ch <- fmt.Sprintf("[-] "+format, args...)
}

func logBanner(ch chan<- string, title string) {
	line := "###############################################################"
	ch <- ""
	ch <- line
	ch <- "# " + title
	ch <- line
}
