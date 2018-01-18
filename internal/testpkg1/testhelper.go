package testpkg1

import "github.com/pkg/errors"

func Foo() error {
	return Bar()
}

func Bar() error {
	return errors.New("Test")
}
