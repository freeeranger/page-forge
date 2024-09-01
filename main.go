package main

import (
	"fmt"
	"os"
)

func validateProject() bool {
	path := "."

	// Validate site.json
	info, err := os.Stat(fmt.Sprintf("%s/site.json", path))
	if err != nil || info.IsDir() {
		fmt.Println("ERROR: site.json does not exist")
		return false
	}

	// Validate pages directory
	info, err = os.Stat(fmt.Sprintf("%s/pages", path))
	if err != nil || !info.IsDir() {
		fmt.Println("ERROR: Directory pages not found")
		return false
	}

	items, err := os.ReadDir(fmt.Sprintf("%s/pages", path))
	if err != nil {
		fmt.Println("ERROR: Failed to read contents of pages directory")
		return false
	}

	valid := false
	for _, item := range items {
		if item.Name() == "index.md" && !item.IsDir() {
			valid = true
		}
	}
	if !valid {
		fmt.Println("ERROR: No index.md in pages directory")
		return false
	}

	return true
}

func initProject() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: page-forge init <project-name>")
		return
	}
}

func checkProject() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: page-forge check")
		return
	}
}

func main() {
	error := false
	if len(os.Args) < 2 {
		error = true
	} else {
		switch os.Args[1] {
		case "init":
			initProject()
			break
		case "check":
			checkProject()
			break
		case "build":
			BuildProject()
			break
		default:
			error = true
		}
	}

	if error {
		fmt.Println("Usage: page-forge <command>")
		fmt.Println("Available commands:")
		fmt.Println("  init         Initializes a project")
		fmt.Println("  check        Validates a project")
		fmt.Println("  build        Builds the site")
		return
	}
}
