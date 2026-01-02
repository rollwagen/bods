//go:build windows

package pasteboard

func initialize() error { return nil }

func readText() string {
	return ""
}

func readFileURL() string {
	return ""
}

func readData() (buf []byte, err error) {
	return []byte{}, nil
}

func readImage() (buf []byte, err error) {
	return []byte{}, nil
}

func getType() string {
	return ""
}

func readAllFileURLs() string {
	return ""
}
