package firewall

import "fmt"

type Error struct {
	Code    string
	Message string
}

func (e Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func errorCode(err error, fallback string) string {
	if err == nil {
		return ""
	}
	if fwErr, ok := err.(Error); ok {
		return fwErr.Code
	}
	return fallback
}
