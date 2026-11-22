package refuse

import "strings"

type Error struct {
	Msg     string
	Details []string
}

func (e *Error) Error() string {
	if len(e.Details) == 0 {
		return e.Msg
	}
	return e.Msg + ": " + strings.Join(e.Details, "; ")
}

func New(msg string, details ...string) *Error {
	return &Error{Msg: msg, Details: details}
}
