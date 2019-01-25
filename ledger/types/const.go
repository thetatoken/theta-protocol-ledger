package types

const (
	// DenomThetaWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomThetaWei string = "ThetaWei"

	// DenomTFuelWei is the basic unit of theta, 1 Theta = 10^18 ThetaWei
	DenomTFuelWei string = "TFuelWei"

	// MinimumGasPrice is the minimum gas price for a smart contract transaction
	MinimumGasPrice uint64 = 1e8

	// MinimumTransactionFeeTFuelWei specifies the minimum fee for a regular transaction
	MinimumTransactionFeeTFuelWei uint64 = 1e12

	// MaxAccountsAffectedPerTx specifies the max number of accounts one transaction is allowed to modify to avoid spamming
	MaxAccountsAffectedPerTx = 10000
)

const (
	// ValidatorThetaGenerationRateNumerator is used for calculating the generation rate of Theta for validators
	//ValidatorThetaGenerationRateNumerator int64 = 317
	ValidatorThetaGenerationRateNumerator int64 = 0 // ZERO inflation for Theta

	// ValidatorThetaGenerationRateDenominator is used for calculating the generation rate of Theta for validators
	// ValidatorThetaGenerationRateNumerator / ValidatorThetaGenerationRateDenominator is the amount of ThetaWei
	// generated per existing ThetaWei per new block
	ValidatorThetaGenerationRateDenominator int64 = 1e11

	// ValidatorTFuelGenerationRateNumerator is used for caluclating the generation rate of TFuel for validators
	ValidatorTFuelGenerationRateNumerator int64 = 0 // ZERO initial inflation for TFuel

	// ValidatorTFuelGenerationRateDenominator is used for caluclating the generation rate of TFuel for validators
	// ValidatorTFuelGenerationRateNumerator / ValidatorTFuelGenerationRateDenominator is the amount of TFuelWei
	// generated per existing ThetaWei per new block
	ValidatorTFuelGenerationRateDenominator int64 = 1e9

	// RegularTFuelGenerationRateNumerator is used for caluclating the generation rate of TFuel for other types of accounts
	//RegularTFuelGenerationRateNumerator int64 = 1900
	RegularTFuelGenerationRateNumerator int64 = 0 // ZERO initial inflation for TFuel

	// RegularTFuelGenerationRateDenominator is used for caluclating the generation rate of TFuel for other types of accounts
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
