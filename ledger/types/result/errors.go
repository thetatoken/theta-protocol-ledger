package result

const (
	CodeType_OK CodeType = 0
	// General response codes, 0 ~ 99
	CodeType_InternalError     CodeType = 1
	CodeType_EncodingError     CodeType = 2
	CodeType_BadNonce          CodeType = 3
	CodeType_Unauthorized      CodeType = 4
	CodeType_InsufficientFunds CodeType = 5
	CodeType_UnknownRequest    CodeType = 6
	// Reserved for basecoin, 100 ~ 199
	CodeType_BaseDuplicateAddress     CodeType = 101
	CodeType_BaseEncodingError        CodeType = 102
	CodeType_BaseInsufficientFees     CodeType = 103
	CodeType_BaseInsufficientFunds    CodeType = 104
	CodeType_BaseInsufficientGasPrice CodeType = 105
	CodeType_BaseInvalidInput         CodeType = 106
	CodeType_BaseInvalidOutput        CodeType = 107
	CodeType_BaseInvalidPubKey        CodeType = 108
	CodeType_BaseInvalidSequence      CodeType = 109
	CodeType_BaseInvalidSignature     CodeType = 110
	CodeType_BaseUnknownAddress       CodeType = 111
	CodeType_BaseUnknownPubKey        CodeType = 112
	CodeType_BaseUnknownPlugin        CodeType = 113
	// Reserved for governance, 200 ~ 299
	CodeType_GovUnknownEntity      CodeType = 201
	CodeType_GovUnknownGroup       CodeType = 202
	CodeType_GovUnknownProposal    CodeType = 203
	CodeType_GovDuplicateGroup     CodeType = 204
	CodeType_GovDuplicateMember    CodeType = 205
	CodeType_GovDuplicateProposal  CodeType = 206
	CodeType_GovDuplicateVote      CodeType = 207
	CodeType_GovInvalidMember      CodeType = 208
	CodeType_GovInvalidVote        CodeType = 209
	CodeType_GovInvalidVotingPower CodeType = 210
)

var (
	OK = NewResultOK(nil, "")

	ErrInternalError     = NewError(CodeType_InternalError, "Internal error")
	ErrEncodingError     = NewError(CodeType_EncodingError, "Encoding error")
	ErrBadNonce          = NewError(CodeType_BadNonce, "Error bad nonce")
	ErrUnauthorized      = NewError(CodeType_Unauthorized, "Unauthorized")
	ErrInsufficientFunds = NewError(CodeType_InsufficientFunds, "Insufficient funds")
	ErrUnknownRequest    = NewError(CodeType_UnknownRequest, "Unknown request")

	ErrBaseDuplicateAddress     = NewError(CodeType_BaseDuplicateAddress, "Error (base) duplicate address")
	ErrBaseEncodingError        = NewError(CodeType_BaseEncodingError, "Error (base) encoding error")
	ErrBaseInsufficientFees     = NewError(CodeType_BaseInsufficientFees, "Error (base) insufficient fees")
	ErrBaseInsufficientFunds    = NewError(CodeType_BaseInsufficientFunds, "Error (base) insufficient funds")
	ErrBaseInsufficientGasPrice = NewError(CodeType_BaseInsufficientGasPrice, "Error (base) insufficient gas price")
	ErrBaseInvalidInput         = NewError(CodeType_BaseInvalidInput, "Error (base) invalid input")
	ErrBaseInvalidOutput        = NewError(CodeType_BaseInvalidOutput, "Error (base) invalid output")
	ErrBaseInvalidPubKey        = NewError(CodeType_BaseInvalidPubKey, "Error (base) invalid pubkey")
	ErrBaseInvalidSequence      = NewError(CodeType_BaseInvalidSequence, "Error (base) invalid sequence")
	ErrBaseInvalidSignature     = NewError(CodeType_BaseInvalidSignature, "Error (base) invalid signature")
	ErrBaseUnknownAddress       = NewError(CodeType_BaseUnknownAddress, "Error (base) unknown address")
	ErrBaseUnknownPlugin        = NewError(CodeType_BaseUnknownPlugin, "Error (base) unknown plugin")
	ErrBaseUnknownPubKey        = NewError(CodeType_BaseUnknownPubKey, "Error (base) unknown pubkey")
)
