package connection

import (
	"errors"
	"fmt"
)

// ProtocolError marks peer input that violates P2P framing or message rules.
// Callers can distinguish it from an ordinary transport failure and avoid
// immediately reconnecting to the offending peer.
type ProtocolError struct {
	Err error
}

func (err *ProtocolError) Error() string {
	return fmt.Sprintf("p2p protocol violation: %v", err.Err)
}

func (err *ProtocolError) Unwrap() error {
	return err.Err
}

func markProtocolError(err error) error {
	if err == nil {
		err = errors.New("invalid peer input")
	}
	var protocolErr *ProtocolError
	if errors.As(err, &protocolErr) {
		return err
	}
	return &ProtocolError{Err: err}
}

// IsProtocolError reports whether a connection error was caused by invalid
// peer-controlled protocol data rather than an ordinary transport failure.
func IsProtocolError(reason interface{}) bool {
	err, ok := reason.(error)
	if !ok {
		return false
	}
	var protocolErr *ProtocolError
	return errors.As(err, &protocolErr)
}
