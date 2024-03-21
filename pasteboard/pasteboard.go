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

// Read returns the bytes of the clipboard data, or nil.
func Read() []byte {
	lock.Lock()
	defer lock.Unlock()

	buf, err := readImage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read clipboard error: %v\n", err)
		return nil
	}
	return buf
}
