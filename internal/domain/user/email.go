package user

import "strings"

type Email string

func NewEmail(raw string) Email {
	return Email(strings.ToLower(strings.TrimSpace(raw)))
}

func (e Email) String() string {
	return string(e)
}
