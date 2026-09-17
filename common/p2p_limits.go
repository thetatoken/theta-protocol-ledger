package common

import "time"

const (
	// MaxBlockMessageSize is the receive limit for block-sized consensus payloads.
	MaxBlockMessageSize = 12 * 1024 * 1024
	// MaxNormalMessageSize is the receive limit for all non-block P2P payloads.
	MaxNormalMessageSize = 1024 * 1024
	// MinP2PReassemblyRate bounds how long a peer may occupy a receive
	// buffer while still allowing slow, continuously progressing transfers.
	MinP2PReassemblyRate = 64 * 1024
)

func MaxP2PMessageSize(channelID ChannelIDEnum) int {
	if channelID == ChannelIDBlock || channelID == ChannelIDProposal {
		return MaxBlockMessageSize
	}
	return MaxNormalMessageSize
}

// IsP2PMessageSizeAllowed reports whether a fully encoded message is valid for
// the given channel.
func IsP2PMessageSizeAllowed(channelID ChannelIDEnum, size int) bool {
	return size >= 0 && size <= MaxP2PMessageSize(channelID)
}

// MaxP2PReassemblyDuration combines an inactivity grace period with enough
// time to receive the largest allowed message at the minimum accepted rate.
func MaxP2PReassemblyDuration(maxMessageSize int, inactivityTimeout time.Duration) time.Duration {
	if inactivityTimeout <= 0 {
		return 0
	}
	if maxMessageSize <= 0 {
		maxMessageSize = MaxNormalMessageSize
	}
	transferNanos := (int64(maxMessageSize)*int64(time.Second) + MinP2PReassemblyRate - 1) /
		MinP2PReassemblyRate
	return inactivityTimeout + time.Duration(transferNanos)
}

// P2PReassemblyDeadline returns the earlier of the inactivity and absolute
// reassembly deadlines. A zero start time means no message is in progress.
func P2PReassemblyDeadline(startedAt, lastProgressAt time.Time, maxMessageSize int,
	inactivityTimeout time.Duration) time.Time {
	if startedAt.IsZero() || inactivityTimeout <= 0 {
		return time.Time{}
	}
	if lastProgressAt.IsZero() {
		lastProgressAt = startedAt
	}
	inactivityDeadline := lastProgressAt.Add(inactivityTimeout)
	absoluteDeadline := startedAt.Add(MaxP2PReassemblyDuration(maxMessageSize, inactivityTimeout))
	if absoluteDeadline.Before(inactivityDeadline) {
		return absoluteDeadline
	}
	return inactivityDeadline
}
