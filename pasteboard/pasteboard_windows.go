//go:build windows

package pasteboard

func initialize() error { return nil }

func readImage() (buf []byte, err error) {
	return []byte{}, nil
}
