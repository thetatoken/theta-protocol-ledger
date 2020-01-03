package common

// HeightEnableValidatorReward specifies the minimal block height to enable the validtor TFUEL reward
const HeightEnableValidatorReward uint64 = 4164982 // approximate time: 2pm January 14th, 2020

// CheckpointInterval defines the interval between checkpoints.
const CheckpointInterval = int64(100)

// IsCheckPointHeight returns if a block height is a checkpoint.
func IsCheckPointHeight(height uint64) bool {
	return height%uint64(CheckpointInterval) == 1
}
