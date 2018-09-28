package result

type ErrorCode int

const (
	CodeOK ErrorCode = 0

	CodeGenericError ErrorCode = 10000
	CodeInvalidNonce ErrorCode = 10001
)
