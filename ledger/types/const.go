package types

import (
	"math/big"

	"github.com/thetatoken/theta/common"
)

const (
	// DenomThetaWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomThetaWei string = "ThetaWei"

	// DenomTFuelWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomTFuelWei string = "TFuelWei"

	// Initial gas parameters

	// MinimumGasPrice is the minimum gas price for a smart contract transaction
	MinimumGasPrice uint64 = 1e8

	// MaximumTxGasLimit is the maximum gas limit for a smart contract transaction
	//MaximumTxGasLimit uint64 = 2e6
	MaximumTxGasLimit uint64 = 10e6

	// MinimumTransactionFeeTFuelWei specifies the minimum fee for a regular transaction
	MinimumTransactionFeeTFuelWei uint64 = 1e12

	// June 2021 gas burn adjustment

	// MinimumGasPrice is the minimum gas price for a smart contract transaction
	MinimumGasPriceJune2021 uint64 = 4e12

	// MaximumTxGasLimit is the maximum gas limit for a smart contract transaction
	MaximumTxGasLimitJune2021 uint64 = 20e6

	// MinimumTransactionFeeTFuelWei specifies the minimum fee for a regular transaction
	MinimumTransactionFeeTFuelWeiJune2021 uint64 = 3e17

	// MaxAccountsAffectedPerTx specifies the max number of accounts one transaction is allowed to modify to avoid spamming
	MaxAccountsAffectedPerTx = 512
)

const (
	// ValidatorThetaGenerationRateNumerator is used for calculating the generation rate of Theta for validators
	//ValidatorThetaGenerationRateNumerator int64 = 317
	ValidatorThetaGenerationRateNumerator int64 = 0 // ZERO inflation for Theta

	// ValidatorThetaGenerationRateDenominator is used for calculating the generation rate of Theta for validators
	// ValidatorThetaGenerationRateNumerator / ValidatorThetaGenerationRateDenominator is the amount of ThetaWei
	// generated per existing ThetaWei per new block
	ValidatorThetaGenerationRateDenominator int64 = 1e11

	// ValidatorTFuelGenerationRateNumerator is used for calculating the generation rate of TFuel for validators
	ValidatorTFuelGenerationRateNumerator int64 = 0 // ZERO initial inflation for TFuel

	// ValidatorTFuelGenerationRateDenominator is used for calculating the generation rate of TFuel for validators
	// ValidatorTFuelGenerationRateNumerator / ValidatorTFuelGenerationRateDenominator is the amount of TFuelWei
	// generated per existing ThetaWei per new block
	ValidatorTFuelGenerationRateDenominator int64 = 1e9

	// RegularTFuelGenerationRateNumerator is used for calculating the generation rate of TFuel for other types of accounts
	//RegularTFuelGenerationRateNumerator int64 = 1900
	RegularTFuelGenerationRateNumerator int64 = 0 // ZERO initial inflation for TFuel

	// RegularTFuelGenerationRateDenominator is used for calculating the generation rate of TFuel for other types of accounts
	// RegularTFuelGenerationRateNumerator / RegularTFuelGenerationRateDenominator is the amount of TFuelWei
	// generated per existing ThetaWei per new block
	RegularTFuelGenerationRateDenominator int64 = 1e10
)

const (

	// ServiceRewardVerificationBlockDelay gives the block delay for service certificate verification
	ServiceRewardVerificationBlockDelay uint64 = 2

	// ServiceRewardFulfillmentBlockDelay gives the block delay for service reward fulfillment
	ServiceRewardFulfillmentBlockDelay uint64 = 4
)

const (

	// MaximumTargetAddressesForStakeBinding gives the maximum number of target addresses that can be associated with a bound stake
	MaximumTargetAddressesForStakeBinding uint = 1024

	// MaximumFundReserveDuration indicates the maximum duration (in terms of number of blocks) of reserving fund
	MaximumFundReserveDuration uint64 = 12 * 3600

	// MinimumFundReserveDuration indicates the minimum duration (in terms of number of blocks) of reserving fund
	MinimumFundReserveDuration uint64 = 300

	// ReservedFundFreezePeriodDuration indicates the freeze duration (in terms of number of blocks) of the reserved fund
	ReservedFundFreezePeriodDuration uint64 = 5
)

func GetMinimumGasPrice(blockHeight uint64) *big.Int {
	if blockHeight < common.HeightJune2021FeeAdjustment {
		return new(big.Int).SetUint64(MinimumGasPrice)
	}

	return new(big.Int).SetUint64(MinimumGasPriceJune2021)
}

func GetMaxGasLimit(blockHeight uint64) *big.Int {
	if blockHeight < common.HeightJune2021FeeAdjustment {
		return new(big.Int).SetUint64(MaximumTxGasLimit)
	}

	return new(big.Int).SetUint64(MaximumTxGasLimitJune2021)
}

func GetMinimumTransactionFeeTFuelWei(blockHeight uint64) *big.Int {
	if blockHeight < common.HeightJune2021FeeAdjustment {
		return new(big.Int).SetUint64(MinimumTransactionFeeTFuelWei)
	}

	return new(big.Int).SetUint64(MinimumTransactionFeeTFuelWeiJune2021)
}

// Special handling for many-to-many SendTx
func GetSendTxMinimumTransactionFeeTFuelWei(numAccountsAffected uint64, blockHeight uint64) *big.Int {
	if blockHeight < common.HeightJune2021FeeAdjustment {
		return new(big.Int).SetUint64(MinimumTransactionFeeTFuelWei) // backward compatiblity
	}

	if numAccountsAffected < 2 {
		numAccountsAffected = 2
	}

	// minSendTxFee = numAccountsAffected * MinimumTransactionFeeTFuelWeiJune2021 / 2
	minSendTxFee := big.NewInt(1).Mul(new(big.Int).SetUint64(numAccountsAffected), new(big.Int).SetUint64(MinimumTransactionFeeTFuelWeiJune2021))
	minSendTxFee = big.NewInt(1).Div(minSendTxFee, new(big.Int).SetUint64(2))

	return minSendTxFee
}
