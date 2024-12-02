package main

import (
	"os"
	"slices"
	"strings"
)

// Detect some files or folders and ignore some files or folders
// type IgnoreRule struct {
// 	checkList func([]os.DirEntry) []string
// }

type StIgnoreCheckFunc = func([]os.DirEntry) []string

var StIgnoreCheckList = []StIgnoreCheckFunc{
	RustProjectStIgnoreChecker,
	NodejsProjectStIgnoreChecker,
	DartProjectStIgnoreChecker,
	PythonCondaStIgnoreChecker,
}

// Ignore Rust build files
// If it contains Cargo.toml and Cargo.lock, it is considered a Rust project
var RustProjectStIgnoreChecker = func(entry []os.DirEntry) []string {
	var filenames = make([]string, 0)
	for _, v := range entry {
		filenames = append(filenames, v.Name())
	}
	if slices.Contains(filenames, "Cargo.toml") && slices.Contains(filenames, "Cargo.lock") {
		return []string{"target"}
	}
	return nil
}

// Ignore Node.js project
// If it contains package.json and node_modules, it is considered a Node.js project
var NodejsProjectStIgnoreChecker = func(entry []os.DirEntry) []string {
	var filenames = make([]string, 0)
	for _, v := range entry {
		filenames = append(filenames, v.Name())
	}
	if slices.Contains(filenames, "package.json") && slices.Contains(filenames, "node_modules") {
		return []string{"node_modules", "dist"}
	}
	return nil
}

// Ignore Flutter project
// If it contains pubspec.yaml and pubspec.lock, it is considered a Flutter project
var DartProjectStIgnoreChecker = func(entry []os.DirEntry) []string {
	var filenames = make([]string, 0)
	for _, v := range entry {
		filenames = append(filenames, v.Name())
	}
	if slices.Contains(filenames, "pubspec.yaml") && slices.Contains(filenames, "pubspec.lock") {
		return []string{"build"}
	}
	return nil
}

// Ignore Python .conda
var PythonCondaStIgnoreChecker = func(entry []os.DirEntry) []string {
	var filenames = make([]string, 0)
	for _, v := range entry {
		filenames = append(filenames, v.Name())
	}
	for _, v := range filenames {
		if strings.HasPrefix(v, ".conda") {
			return []string{".conda"}
		}
	}
	return nil
}
