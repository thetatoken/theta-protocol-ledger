package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database/backend"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "genesis"})

const (
	GenBlockHashMode int = iota
	GenGenesisFileMode
)

type StakeDeposit struct {
	Source string `json:"source"`
	Holder string `json:"holder"`
	Amount string `json:"amount"`
}

//
// Example:
// cd $THETA/integration/privatenet/node
// generate_genesis -mode=0 -chainID=private_net -erc20snapshot=./data/genesis_theta_erc20_snapshot.json -stake_deposit=./data/genesis_stake_deposit.json -genesis_signatures=./data/genesis_signatures.json -genesis=./genesis
//
func main() {
	mode, chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath, genesisSignaturesFilePath, genesisSnapshotFilePath := parseArguments()

	sv, metadata, err := generateGenesisSnapshot(mode, chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath, genesisSignaturesFilePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate genesis snapshot: %v", err))
	}

	err = sanityChecks(sv)
	if err != nil {
		panic(fmt.Sprintf("Sanity checks failed: %v", err))
	}

	if mode == GenGenesisFileMode {
		err = writeGenesisSnapshot(sv, metadata, genesisSnapshotFilePath)
		if err != nil {
			panic(fmt.Sprintf("Failed to write genesis snapshot: %v", err))
		}
	}
}

func parseArguments() (mode int, chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath,
	genesisSignaturesFilePath, genesisSnapshotFilePath string) {
	modePtr := flag.Int("mode", GenBlockHashMode, "0: print the hash of the second block, 1: generate the genesis file")
	chainIDPtr := flag.String("chainID", "local_chain", "the ID of the chain")
	erc20SnapshotJSONFilePathPtr := flag.String("erc20snapshot", "./theta_erc20_snapshot.json", "the json file contain the ERC20 balance snapshot")
	stakeDepositFilePathPtr := flag.String("stake_deposit", "./stake_deposit.json", "the initial stake deposits")
	genesisSignaturesFilePathPtr := flag.String("genesis_signatures", "./genesis_signatures.json", "the genesis signatures")
	genesisSnapshotFilePathPtr := flag.String("genesis", "./genesis", "the genesis snapshot")
	flag.Parse()

	mode = *modePtr
	chainID = *chainIDPtr
	erc20SnapshotJSONFilePath = *erc20SnapshotJSONFilePathPtr
	stakeDepositFilePath = *stakeDepositFilePathPtr
	genesisSignaturesFilePath = *genesisSignaturesFilePathPtr
	genesisSnapshotFilePath = *genesisSnapshotFilePathPtr

	return
}

// generateGenesisSnapshot generates the genesis snapshot.
func generateGenesisSnapshot(mode int, chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath,
	genesisSignaturesFilePath string) (*state.StoreView, *core.SnapshotMetadata, error) {
	metadata := &core.SnapshotMetadata{}
	genesisHeight := uint64(0)

	sv := loadInitialBalances(erc20SnapshotJSONFilePath)
	vcp := performInitialStakeDeposit(stakeDepositFilePath, genesisHeight, sv)

	// --------------- Generate Genensis Blocks --------------- //

	stateHash := sv.Hash()
	validators := consensus.SelectTopStakeHoldersAsValidators(vcp).Validators()

	// First block

	firstBlock := core.NewBlock()
	firstBlock.ChainID = chainID
	firstBlock.Height = genesisHeight
	firstBlock.Epoch = firstBlock.Height
	firstBlock.Parent = common.Hash{}
	firstBlock.StateHash = stateHash
	firstBlock.Timestamp = big.NewInt(time.Now().Unix())

	// Second block

	secondBlock := core.NewBlock()
	secondBlock.ChainID = chainID
	secondBlock.Height = genesisHeight + 1
	secondBlock.Epoch = secondBlock.Height
	secondBlock.Parent = firstBlock.Hash()
	secondBlock.HCC = core.CommitCertificate{BlockHash: firstBlock.Hash()}
	secondBlock.StateHash = stateHash
	secondBlock.Timestamp = big.NewInt(time.Now().Unix())

	// Third block (dummy block)

	thirdBlock := core.NewBlock()
	thirdBlock.Parent = secondBlock.Hash()
	thirdBlock.HCC = core.CommitCertificate{BlockHash: secondBlock.Hash()}

	if mode == GenBlockHashMode {
		fmt.Printf("\n")
		fmt.Printf("Block hash to be signed: %v\n", thirdBlock.Hash())
		fmt.Printf("\n")
		return sv, nil, nil
	}

	genesisSignaturesFile, err := os.Open(genesisSignaturesFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the genesis signatures file: %v", err))
	}
	defer genesisSignaturesFile.Close()

	var genesisSignatureMap map[string][]byte
	genesisSignaturesMapByteValue, err := ioutil.ReadAll(genesisSignaturesFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read genesis signatures file: %v", err))
	}

	json.Unmarshal(genesisSignaturesMapByteValue, &genesisSignatureMap)

	thirdBlockVoteSet := core.NewVoteSet()
	for _, validator := range validators {
		sigBytes := genesisSignatureMap[validator.Address.Hex()]
		sig, err := crypto.SignatureFromBytes(sigBytes)
		if err != nil {
			panic(fmt.Sprintf("failed to get genesis signature: %v", err))
		}

		thirdBlockVote := core.Vote{
			Block:  thirdBlock.Hash(),
			Height: thirdBlock.Height,
			ID:     validator.Address,
			Epoch:  thirdBlock.Height,
		}
		thirdBlockVote.SetSignature(sig)
		thirdBlockVoteSet.AddVote(thirdBlockVote)
	}

	vcpProof, err := proveVCP(sv)
	if err != nil {
		panic(fmt.Errorf("Failed to get VCP Proof"))
	}

	metadata.BlockTrios = append(metadata.BlockTrios,
		core.SnapshotBlockTrio{
			First:  core.SnapshotFirstBlock{Header: *firstBlock.BlockHeader, Proof: *vcpProof},
			Second: core.SnapshotSecondBlock{Header: *secondBlock.BlockHeader},
			Third:  core.SnapshotThirdBlock{Header: *thirdBlock.BlockHeader, VoteSet: thirdBlockVoteSet},
		})

	return sv, metadata, nil
}

func loadInitialBalances(erc20SnapshotJSONFilePath string) *state.StoreView {
	initTFuelToThetaRatio := new(big.Int).SetUint64(5)
	sv := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())

	erc20SnapshotJSONFile, err := os.Open(erc20SnapshotJSONFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the ERC20 balance snapshot: %v", err))
	}
	defer erc20SnapshotJSONFile.Close()

	var erc20BalanceMap map[string]string
	erc20BalanceMapByteValue, err := ioutil.ReadAll(erc20SnapshotJSONFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal(erc20BalanceMapByteValue, &erc20BalanceMap)
	for key, val := range erc20BalanceMap {
		if !common.IsHexAddress(key) {
			panic(fmt.Sprintf("Invalid address: %v", key))
		}
		address := common.HexToAddress(key)

		theta, success := new(big.Int).SetString(val, 10)
		if !success {
			panic(fmt.Sprintf("Failed to parse ThetaWei amount: %v", val))
		}
		tfuel := new(big.Int).Mul(initTFuelToThetaRatio, theta)
		acc := &types.Account{
			Address: address,
			Balance: types.Coins{
				ThetaWei: theta,
				TFuelWei: tfuel,
			},
			LastUpdatedBlockHeight: 0,
		}
		sv.SetAccount(acc.Address, acc)
		//logger.Infof("address: %v, theta: %v, tfuel: %v", strings.ToLower(address.String()), theta, tfuel)
	}

	return sv
}

func performInitialStakeDeposit(stakeDepositFilePath string, genesisHeight uint64, sv *state.StoreView) *core.ValidatorCandidatePool {
	var stakeDeposits []StakeDeposit
	stakeDepositFile, err := os.Open(stakeDepositFilePath)
	stakeDepositByteValue, err := ioutil.ReadAll(stakeDepositFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read initial stake deposit file: %v", err))
	}

	json.Unmarshal(stakeDepositByteValue, &stakeDeposits)
	vcp := &core.ValidatorCandidatePool{}
	for _, stakeDeposit := range stakeDeposits {
		if !common.IsHexAddress(stakeDeposit.Source) {
			panic(fmt.Sprintf("Invalid source address: %v", stakeDeposit.Source))
		}
		if !common.IsHexAddress(stakeDeposit.Holder) {
			panic(fmt.Sprintf("Invalid holder address: %v", stakeDeposit.Holder))
		}
		sourceAddress := common.HexToAddress(stakeDeposit.Source)
		holderAddress := common.HexToAddress(stakeDeposit.Holder)
		stakeAmount, success := new(big.Int).SetString(stakeDeposit.Amount, 10)
		if !success {
			panic(fmt.Sprintf("Failed to parse Stake amount: %v", stakeDeposit.Amount))
		}

		sourceAccount := sv.GetAccount(sourceAddress)
		if sourceAccount == nil {
			panic(fmt.Sprintf("Failed to retrieve account for source address: %v", sourceAddress))
		}
		if sourceAccount.Balance.ThetaWei.Cmp(stakeAmount) < 0 {
			panic(fmt.Sprintf("The source account %v does NOT have sufficient balance for stake deposit. ThetaWeiBalance = %v, StakeAmount = %v",
				sourceAddress, sourceAccount.Balance.ThetaWei, stakeDeposit.Amount))
		}
		err := vcp.DepositStake(sourceAddress, holderAddress, stakeAmount)
		if err != nil {
			panic(fmt.Sprintf("Failed to deposit stake, err: %v", err))
		}

		stake := types.Coins{
			ThetaWei: stakeAmount,
			TFuelWei: new(big.Int).SetUint64(0),
		}
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		sv.SetAccount(sourceAddress, sourceAccount)
	}

	sv.UpdateValidatorCandidatePool(vcp)

	hl := &types.HeightList{}
	hl.Append(genesisHeight)
	sv.UpdateStakeTransactionHeightList(hl)

	return vcp
}

func proveVCP(sv *state.StoreView) (*core.VCPProof, error) {
	vp := &core.VCPProof{}
	vcpKey := state.ValidatorCandidatePoolKey()
	err := sv.ProveVCP(vcpKey, vp)
	return vp, err
}

// writeGenesisSnapshot writes genesis snapshot to file system.
func writeGenesisSnapshot(sv *state.StoreView, metadata *core.SnapshotMetadata, genesisSnapshotFilePath string) error {
	file, err := os.Create(genesisSnapshotFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return err
	}

	writeStoreView(sv, true, writer)
	return err
}

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer) {
	height := core.Itobytes(sv.Height())
	err := core.WriteRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = core.WriteRecord(writer, k, v)
		if err != nil {
			panic(err)
		}
		return true
	})
	err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func sanityChecks(sv *state.StoreView) error {
	thetaWeiTotal := new(big.Int).SetUint64(0)
	tfuelWeiTotal := new(big.Int).SetUint64(0)

	vcpAnalyzed := false
	sv.GetStore().Traverse(nil, func(key, val common.Bytes) bool {
		if bytes.Compare(key, state.ValidatorCandidatePoolKey()) == 0 {
			var vcp core.ValidatorCandidatePool
			err := rlp.DecodeBytes(val, &vcp)
			if err != nil {
				panic(fmt.Sprintf("Failed to decode VCP: %v", err))
			}
			for _, sc := range vcp.SortedCandidates {
				logger.Infof("--------------------------------------------------------")
				logger.Infof("Validator Candidate: %v, totalStake  = %v", sc.Holder, sc.TotalStake())
				for _, stake := range sc.Stakes {
					thetaWeiTotal = new(big.Int).Add(thetaWeiTotal, stake.Amount)
					logger.Infof("     Stake: source = %v, stakeAmount = %v", stake.Source, stake.Amount)
				}
				logger.Infof("--------------------------------------------------------")
			}
			vcpAnalyzed = true
		} else if bytes.Compare(key, state.StakeTransactionHeightListKey()) == 0 {
		} else { // regular account
			var account types.Account
			err := rlp.DecodeBytes(val, &account)
			if err != nil {
				panic(fmt.Sprintf("Failed to decode Account: %v", err))
			}

			thetaWei := account.Balance.ThetaWei
			tfuelWei := account.Balance.TFuelWei
			thetaWeiTotal = new(big.Int).Add(thetaWeiTotal, thetaWei)
			tfuelWeiTotal = new(big.Int).Add(tfuelWeiTotal, tfuelWei)

			logger.Infof("Account: %v, ThetaWei = %v, TFuelWei = %v", account.Address, thetaWei, tfuelWei)
		}
		return true
	})

	// Check #1: VCP analyzed
	if !vcpAnalyzed {
		return fmt.Errorf("VCP not detected in the genesis file")
	}

	// Check #2: Sum(ThetaWei) + Sum(Stake) == 1 * 10^9 * 10^18
	oneBillion := new(big.Int).SetUint64(1000000000)
	fiveBillion := new(big.Int).Mul(new(big.Int).SetUint64(5), oneBillion)
	ten18 := new(big.Int).SetUint64(1000000000000000000)

	expectedThetaWeiTotal := new(big.Int).Mul(oneBillion, ten18)
	if expectedThetaWeiTotal.Cmp(thetaWeiTotal) != 0 {
		return fmt.Errorf("Unmatched ThetaWei total: expected = %v, calculated = %v", expectedThetaWeiTotal, thetaWeiTotal)
	}
	logger.Infof("Expected   ThetaWei total = %v", expectedThetaWeiTotal)
	logger.Infof("Calculated ThetaWei total = %v", thetaWeiTotal)

	// Check #3: Sum(TFuelWei) == 5 * 10^9 * 10^18
	expectedTFuelWeiTotal := new(big.Int).Mul(fiveBillion, ten18)
	if expectedTFuelWeiTotal.Cmp(tfuelWeiTotal) != 0 {
		return fmt.Errorf("Unmatched TFuelWei total: expected = %v, calculated = %v", expectedTFuelWeiTotal, tfuelWeiTotal)
	}
	logger.Infof("Expected   TFuelWei total = %v", expectedTFuelWeiTotal)
	logger.Infof("Calculated TFuelWei total = %v", tfuelWeiTotal)

	return nil
}
