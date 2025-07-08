package combinederror

import "strings"

var _ error = &combinedError{}

type combinedError struct {
	errors []error
}

func NewCombinedError() *combinedError {
	return &combinedError{}
}

func (c *combinedError) Error() string {
	var msgs []string
	for _, err := range c.errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

func (c *combinedError) Append(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			c.errors = append(c.errors, err)
		}
	}
	return c
}
