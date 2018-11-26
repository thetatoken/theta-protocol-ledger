package types

const (
	// DenomThetaWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomThetaWei string = "ThetaWei"

	// DenomGammaWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomGammaWei string = "GammaWei"

	// MinimumGasPrice is the minimum gas price for a smart contract transaction
	MinimumGasPrice uint64 = 1e8

	// MinimumTransactionFeeGammaWei specifies the minimum fee for a regular transaction
	MinimumTransactionFeeGammaWei uint64 = 1e12
)

const (
	// ValidatorThetaGenerationRateNumerator is used for calculating the generation rate of Theta for validators
	//ValidatorThetaGenerationRateNumerator int64 = 317
	ValidatorThetaGenerationRateNumerator int64 = 0 // ZERO inflation for Theta

	// ValidatorThetaGenerationRateDenominator is used for calculating the generation rate of Theta for validators
	// ValidatorThetaGenerationRateNumerator / ValidatorThetaGenerationRateDenominator is the amount of ThetaWei
	// generated per existing ThetaWei per new block
	ValidatorThetaGenerationRateDenominator int64 = 1e11

	// ValidatorGammaGenerationRateNumerator is used for caluclating the generation rate of Gamma for validators
	ValidatorGammaGenerationRateNumerator int64 = 0 // ZERO initial inflation for Gamma

	// ValidatorGammaGenerationRateDenominator is used for caluclating the generation rate of Gamma for validators
	// ValidatorGammaGenerationRateNumerator / ValidatorGammaGenerationRateDenominator is the amount of GammaWei
	// generated per existing ThetaWei per new block
	ValidatorGammaGenerationRateDenominator int64 = 1e9

	// RegularGammaGenerationRateNumerator is used for caluclating the generation rate of Gamma for other types of accounts
	//RegularGammaGenerationRateNumerator int64 = 1900
	RegularGammaGenerationRateNumerator int64 = 0 // ZERO initial inflation for Gamma

	// RegularGammaGenerationRateDenominator is used for caluclating the generation rate of Gamma for other types of accounts
	// RegularGammaGenerationRateNumerator / RegularGammaGenerationRateDenominator is the amount of GammaWei
	// generated per existing ThetaWei per new block
	RegularGammaGenerationRateDenominator int64 = 1e10
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
