package testhelper

import (
	"errors"

	pkgerrors "github.com/pkg/errors"
)

func NewError() error {
	err := errors.New("test error")
	return pkgerrors.Wrap(err, "")
}
