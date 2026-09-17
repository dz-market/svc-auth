package user

import "errors"

var ErrEmailTaken = errors.New("email is already taken")
