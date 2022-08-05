package common

// // HeightEnableValidatorReward specifies the minimal block height to enable the validtor TFUEL reward
// const HeightEnableValidatorReward uint64 = 4164982 // approximate time: 2pm January 14th, 2020 PST

// // HeightEnableTheta2 specifies the minimal block height to enable the Theta2.0 feature.
// const HeightEnableTheta2 uint64 = 5877350 // approximate time: 12pm May 27th, 2020 PDT

// // HeightLowerGNStakeThresholdTo1000 specifies the minimal block height to lower the GN Stake Threshold to 1,000 THETA
// const HeightLowerGNStakeThresholdTo1000 uint64 = 8411427 // approximate time: 12pm Dec 10th, 2020 PST

// // HeightEnableSmartContract specifies the minimal block height to eanble the Turing-complete smart contract support
// const HeightEnableSmartContract uint64 = 8411427 // approximate time: 12pm Dec 10th, 2020 PST

// // HeightSampleStakingReward specifies the block heigth to enable sampling of staking reward
// const HeightSampleStakingReward uint64 = 9497418 // approximate time: 7pm Mar 10th, 2021 PST

// // HeightJune2021FeeAdjustment specifies the block heigth to enable transaction fee burning adjustment
// const HeightJune2021FeeAdjustment uint64 = 10709540 // approximate time: 12pm June 11, 2021 PT

// // HeightEnableTheta3 specifies the minimal block height to enable the Theta3.0 feature.
// const HeightEnableTheta3 uint64 = 10968061 // approximate time: 12pm June 30, 2021 PT

// // HeightRPCCompatibility specifies the block height to enable Ethereum compatible RPC support
// const HeightRPCCompatibility uint64 = 1000000000

// HeightEnableValidatorReward specifies the minimal block height to enable the validtor TFUEL reward
const HeightEnableValidatorReward uint64 = 2 // approximate time: 2pm January 14th, 2020 PST

// HeightEnableTheta2 specifies the minimal block height to enable the Theta2.0 feature.
const HeightEnableTheta2 uint64 = 2 // approximate time: 12pm May 27th, 2020 PDT

// HeightLowerGNStakeThresholdTo1000 specifies the minimal block height to lower the GN Stake Threshold to 1,000 THETA
const HeightLowerGNStakeThresholdTo1000 uint64 = 2 // approximate time: 12pm Dec 10th, 2020 PST

// HeightEnableSmartContract specifies the minimal block height to eanble the Turing-complete smart contract support
const HeightEnableSmartContract uint64 = 2 // approximate time: 12pm Dec 10th, 2020 PST

// HeightSampleStakingReward specifies the block heigth to enable sampling of staking reward
const HeightSampleStakingReward uint64 = 2 // approximate time: 7pm Mar 10th, 2021 PST

// HeightJune2021FeeAdjustment specifies the block heigth to enable transaction fee burning adjustment
const HeightJune2021FeeAdjustment uint64 = 2 // approximate time: 12pm June 11, 2021 PT

// HeightEnableTheta3 specifies the minimal block height to enable the Theta3.0 feature.
const HeightEnableTheta3 uint64 = 2 // approximate time: 12pm June 30, 2021 PT

// HeightRPCCompatibility specifies the block height to enable Ethereum compatible RPC support
const HeightRPCCompatibility uint64 = 2 // approximate time: 12pm July 30, 2021 PT

// HeightTxWrapperExtension specifies the block height to extend the Tx Wrapper
const HeightTxWrapperExtension uint64 = 2

// HeightSupportThetaTokenInSmartContract specifies the block height to support Theta in smart contracts
const HeightSupportThetaTokenInSmartContract uint64 = 2 // approximate time: 5pm Dec 4, 2021 PT

// HeightValidatorStakeChangedTo200K specifies the block height to lower the validator stake to 200,000 Theta
const HeightValidatorStakeChangedTo200K uint64 = 2

// HeightEnableMetachainSupport specifies the block height to enable Theta Metachain support (i.e. Mainnet 4.0)
const HeightEnableMetachainSupport uint64 = 0

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
