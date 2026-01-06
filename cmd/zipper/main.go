package main

import (
	"flag"
	"fmt"
	"os"

	zipper "github.com/irrisdev/go-zip"
)

func main() {
	// Define flags
	path := flag.String("path", "", "path to file or directory to zip")
	flag.Parse()

	// Validate required flag
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Error: -path flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Compress the path
	zipPath, err := zipper.Zip(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error zipping %s: %v\n", *path, err)
		os.Exit(1)
	}

	fmt.Printf("successfully created: %s\n", zipPath)
}
