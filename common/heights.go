package common

// HeightEnableValidatorReward specifies the minimal block height to enable the validtor TFUEL reward
//const HeightEnableValidatorReward uint64 = 10000000000
const HeightEnableValidatorReward uint64 = 457077 // For Amber testnet

// HeightEnableTheta2 specifies the minimal block height to enable the Theta2.0 feature.
//const HeightEnableTheta2 uint64 = 10000000000
const HeightEnableTheta2 uint64 = 457077 // for Amber testnet

// CheckpointInterval defines the interval between checkpoints.
const CheckpointInterval = int64(100)

// IsCheckPointHeight returns if a block height is a checkpoint.
func IsCheckPointHeight(height uint64) bool {
	return height%uint64(CheckpointInterval) == 1
}

// LastCheckPointHeight returns the height of the last checkpoint
func LastCheckPointHeight(height uint64) uint64 {
	multiple := height / uint64(CheckpointInterval)
	lastCheckpointHeight := uint64(CheckpointInterval)*multiple + 1
	return lastCheckpointHeight
}
