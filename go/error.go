package protocol

import (
	"fmt"
)

type LBError struct {
	Status  uint8
	Code    uint64
	Message string
}

func (e *LBError) Error() string {
	return fmt.Sprintf("longbridge protocol api error, status:%d code:%d message:%s", e.Status, e.Code, e.Message)
}

func NewError(status uint8, code uint64, msg string) error {
	return &LBError{
		Status:  status,
		Code:    code,
		Message: msg,
	}
}
