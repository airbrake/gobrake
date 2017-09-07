package gobrake_test

import "github.com/pkg/errors"

func foo() error {
	return bar()
}

func bar() error {
	return errors.New("Test")
}
