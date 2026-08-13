package common

import (
	"testing"

	rootcmn "github.com/thetatoken/theta/common"
)

func TestMaxMessageSizeForChannel(t *testing.T) {
	tests := []struct {
		name      string
		channelID rootcmn.ChannelIDEnum
		want      int
	}{
		{name: "block", channelID: rootcmn.ChannelIDBlock, want: MaxBlockMessageSize},
		{name: "proposal", channelID: rootcmn.ChannelIDProposal, want: MaxBlockMessageSize},
		{name: "vote", channelID: rootcmn.ChannelIDVote, want: MaxNormalMessageSize},
		{name: "transaction", channelID: rootcmn.ChannelIDTransaction, want: MaxNormalMessageSize},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MaxMessageSizeForChannel(test.channelID); got != test.want {
				t.Fatalf("MaxMessageSizeForChannel(%v) = %v, want %v", test.channelID, got, test.want)
			}
		})
	}
}

func TestIsP2PMessageSizeAllowed(t *testing.T) {
	tests := []struct {
		name      string
		channelID rootcmn.ChannelIDEnum
		size      int
		want      bool
	}{
		{name: "negative", channelID: rootcmn.ChannelIDVote, size: -1, want: false},
		{name: "empty", channelID: rootcmn.ChannelIDVote, size: 0, want: true},
		{name: "normal at limit", channelID: rootcmn.ChannelIDVote, size: MaxNormalMessageSize, want: true},
		{name: "normal above limit", channelID: rootcmn.ChannelIDVote, size: MaxNormalMessageSize + 1, want: false},
		{name: "block at limit", channelID: rootcmn.ChannelIDBlock, size: MaxBlockMessageSize, want: true},
		{name: "block above limit", channelID: rootcmn.ChannelIDBlock, size: MaxBlockMessageSize + 1, want: false},
		{name: "proposal at limit", channelID: rootcmn.ChannelIDProposal, size: MaxBlockMessageSize, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rootcmn.IsP2PMessageSizeAllowed(test.channelID, test.size); got != test.want {
				t.Fatalf("IsP2PMessageSizeAllowed(%v, %v) = %v, want %v", test.channelID, test.size, got, test.want)
			}
		})
	}
}
