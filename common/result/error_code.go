package result

type ErrorCode int

const (
	CodeOK ErrorCode = 0

	CodeGenericError             ErrorCode = 10000
	CodeInvalidSignature         ErrorCode = 10001
	CodeInvalidSequence          ErrorCode = 10002
	CodeInsufficientFund         ErrorCode = 10003
	CodeEmptyPubKeyWithSequence1 ErrorCode = 10004
)
