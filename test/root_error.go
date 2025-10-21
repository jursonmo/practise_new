package main

import (
	"errors"
	"fmt"
)

type MyError struct {
	Code    int
	Message string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func getRootCause(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

func main() {
	originalErr := &MyError{Code: 404, Message: "not found"}
	wrappedErr := fmt.Errorf("wrapper 1: %w", originalErr)
	doubleWrappedErr := fmt.Errorf("wrapper 2: %w", wrappedErr)

	rootCause := getRootCause(doubleWrappedErr)
	fmt.Printf("根因 error: %v\n", rootCause)
	// 输出: 根因 error: code: 404, message: not found
}
