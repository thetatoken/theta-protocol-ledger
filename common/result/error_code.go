package result

type ErrorCode int

const (
	CodeOK ErrorCode = 0

	// Common Errors
	CodeGenericError             ErrorCode = 100000
	CodeInvalidSignature         ErrorCode = 100001
	CodeInvalidSequence          ErrorCode = 100002
	CodeInsufficientFund         ErrorCode = 100003
	CodeEmptyPubKeyWithSequence1 ErrorCode = 100004
	CodeUnauthorizedTx           ErrorCode = 100005
	CodeInvalidFee               ErrorCode = 100006

	// ReserveFund Errors
	CodeReserveFundCheckFailed   ErrorCode = 101001
	CodeReservedFundNotSpecified ErrorCode = 101002
	CodeInvalidFundToReserve     ErrorCode = 101003

	// ReleaseFund Errors
	CodeReleaseFundCheckFailed ErrorCode = 102001

	// ServerPayment Errors
	CodeCheckTransferReservedFundFailed ErrorCode = 103001
)
