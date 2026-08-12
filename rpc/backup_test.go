package rpc

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
)

func TestBackupRPCMethodsDisabledByDefault(t *testing.T) {
	previousValue := viper.GetBool(common.CfgRPCEnableAdminMethods)
	viper.Set(common.CfgRPCEnableAdminMethods, false)
	defer viper.Set(common.CfgRPCEnableAdminMethods, previousValue)

	service := &ThetaRPCService{}
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "snapshot",
			call: func() error {
				return service.BackupSnapshot(&BackupSnapshotArgs{}, &BackupSnapshotResult{})
			},
		},
		{
			name: "chain",
			call: func() error {
				return service.BackupChain(&BackupChainArgs{}, &BackupChainResult{})
			},
		},
		{
			name: "chain correction",
			call: func() error {
				return service.BackupChainCorrection(&BackupChainCorrectionArgs{}, &BackupChainCorrectionResult{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, errRPCAdminMethodsDisabled, test.call())
		})
	}
}

func TestRPCAdminMethodsCanBeEnabledExplicitly(t *testing.T) {
	previousValue := viper.GetBool(common.CfgRPCEnableAdminMethods)
	viper.Set(common.CfgRPCEnableAdminMethods, true)
	defer viper.Set(common.CfgRPCEnableAdminMethods, previousValue)

	require.NoError(t, requireRPCAdminMethodsEnabled())
}
