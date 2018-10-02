package result

import "fmt"

// Result represents the result of a function execution
type Result struct {
	Code    ErrorCode
	Message string
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

// -------------- Constructors -------------- //

// OK represents the success result
var OK = Result{Code: CodeOK}

// Error returns an error result
func Error(msgFormat string, a ...interface{}) Result {
	msg := fmt.Sprintf(msgFormat, a)
	return Result{
		Code:    CodeGenericError,
		Message: msg,
	}
}
