package protocol

const (
	StatusSuccess uint8 = iota
	StatusServerTimeout
	StatusClientTimeout
	StatusBadRequest
	StatusBadResponse
	StatusUnauthenticated
	StatusPermissionDenied
	StatusServerInternalError
	StatusClientInternalError
)
