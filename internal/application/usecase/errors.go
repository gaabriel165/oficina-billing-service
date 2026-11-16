package usecase

import "errors"

var ErrForbidden = errors.New("access to this resource is not allowed")
