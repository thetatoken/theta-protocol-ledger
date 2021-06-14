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

	// SplitRule Errors
	CodeUnauthorizedToUpdateSplitRule ErrorCode = 104001

	// SmartContract Errors
	CodeEVMError               ErrorCode = 105001
	CodeInvalidValueToTransfer ErrorCode = 105002
	CodeInvalidGasPrice        ErrorCode = 105003
	CodeFeeLimitTooHigh        ErrorCode = 105004
	CodeInvalidGasLimit        ErrorCode = 105005

	// Stake Deposit/Withdrawal Errors
	CodeInvalidStakePurpose     ErrorCode = 106001
	CodeInvalidStake            ErrorCode = 106002
	CodeInsufficientStake       ErrorCode = 106003
	CodeNotEnoughBalanceToStake ErrorCode = 106004
	CodeStakeExceedsCap         ErrorCode = 106005
)
