package pasteboard

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

var errUnavailable = errors.New("clipboard not available")

var (
	// use a global lock to guarantee one read at a time
	lock      = sync.Mutex{}
	initOnce  sync.Once
	initError error
)

func Init() error {
	initOnce.Do(func() {
		initError = initialize()
	})
	return initError
}

// ReadImagePNG returns the image bytes of the clipboard data, or nil.
func ReadImagePNG() []byte {
	lock.Lock()
	defer lock.Unlock()

	buf, err := readImage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read clipboard error: %v\n", err)
		return nil
	}
	return buf
}

func GetContentType() string {
	lock.Lock()
	defer lock.Unlock()

	t := getType()
	return convertToMimeType(t)
}

func ReadText() string {
	lock.Lock()
	defer lock.Unlock()

	return readText()
}

// ReadFileURL returns a file URL from the pasteboard; if there are multiple,
// it only returns the first one
func ReadFileURL() string {
	lock.Lock()
	defer lock.Unlock()

	return readFileURL()
}

// ReadAllFileURLs returns all file URLs from the pasteboard, separated by newlines
func ReadAllFileURLs() string {
	lock.Lock()
	defer lock.Unlock()

	return readAllFileURLs()
}

func ReadData() []byte {
	lock.Lock()
	defer lock.Unlock()

	// Get content type to determine which specialized read function to use
	contentType := getType()
	mimeType := convertToMimeType(contentType)

	switch mimeType {
	case "text/uri-list":
		// For file URLs, use the specialized function that resolves file references
		// Use readAllFileURLs to get all file URLs
		fileURLs := readAllFileURLs()
		if fileURLs == "" {
			return nil
		}
		return []byte(fileURLs)
	case "text/plain":
		// For text, use the specialized text reader
		text := readText()
		if text == "" {
			return nil
		}
		return []byte(text)
	case "image/png", "image/jpeg", "image/gif", "image/tiff":
		// For images, use the specialized image reader
		buf, err := readImage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read clipboard image error: %v\n", err)
			return nil
		}
		return buf
	default:
		// For other types, fall back to raw data reading
		buf, err := readData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read clipboard data error: %v\n", err)
			return nil
		}
		return buf
	}
}

// convertToMimeType converts pasteboard types to MIME types
func convertToMimeType(pasteboardType string) string {
	switch pasteboardType {
	case "public.utf8-plain-text":
		return "text/plain" // "text/plain; charset=utf-8"
	case "public.file-url":
		return "text/uri-list"
	case "public.png":
		return "image/png"
	case "public.jpeg":
		return "image/jpeg"
	case "public.gif":
		return "image/gif"
	case "public.html":
		return "text/html"
	case "public.rtf":
		return "application/rtf"
	case "public.tiff":
		return "image/tiff"
	case "public.pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
