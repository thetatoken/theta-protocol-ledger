package result

import "fmt"

type Info map[string]interface{}

// Result represents the result of a function execution
type Result struct {
	Code    ErrorCode
	Message string
	Info    Info
}

// IsOK indicates if the execution succeeded
func (res Result) IsOK() bool {
	return res.Code == CodeOK
}

// IsError indicates if the execution results in an error
func (res Result) IsError() bool {
	return res.Code != CodeOK
}

// String returns the string representation of the result
func (res Result) String() string {
	return fmt.Sprintf("Result{code:%v, message:%v}", res.Code, res.Message)
}

// WithErrorCode attach the error code to the result
func (res Result) WithErrorCode(code ErrorCode) Result {
	res.Code = code
	return res
}

// WithMessage appends the result's message with the given message
func (res Result) WithMessage(message string) Result {
	res.Message += message
	return res
}

// -------------- Constructors -------------- //

// OK represents the success result
var OK = Result{
	Code: CodeOK,
	Info: make(Info),
}

// OKWith returns a success result with extra information
func OKWith(info Info) Result {
	res := Result{
		Code: CodeOK,
		Info: make(Info),
	}
	for k, v := range info {
		res.Info[k] = v
	}
	return res
}

// Error returns an error result
func Error(msgFormat string, a ...interface{}) Result {
	msg := fmt.Sprintf(msgFormat, a...)
	return Result{
		Code:    CodeGenericError,
		Message: msg,
		Info:    make(Info),
	}
}
