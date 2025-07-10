//go:build exclude

package main

import (
	"fmt"

	"github.com/rollwagen/bods/pasteboard"
)

func main() {
	// Initialize pasteboard
	if err := pasteboard.Init(); err != nil {
		fmt.Printf("Failed to initialize pasteboard: %v\n", err)
		return
	}

	// Check content type
	contentType := pasteboard.GetContentType()
	fmt.Printf("Pasteboard content type: %s\n", contentType)

	if contentType == "text/uri-list" {
		// Test reading single file URL (old API)
		singleURL := pasteboard.ReadFileURL()
		fmt.Printf("\nSingle file URL (old API):\n%s\n", singleURL)

		// Test reading all file URLs (new API)
		allURLs := pasteboard.ReadAllFileURLs()
		fmt.Printf("\nAll file URLs (new API):\n%s\n", allURLs)

		// Test ReadData which should now return all URLs
		data := pasteboard.ReadData()
		if data != nil {
			fmt.Printf("\nReadData result:\n%s\n", string(data))
		}
	} else {
		fmt.Println("No file URLs in pasteboard. Copy some files and try again.")
	}
}
