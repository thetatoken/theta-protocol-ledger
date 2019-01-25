package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

//
// Example:
// cd $THETA/integration/privatenet/node
// generate_genesis -chainID=private_net -erc20snapshot=./data/genesis_theta_erc20_snapshot.json -stake_deposit=./data/genesis_stake_deposit.json -genesis_signatures=./data/genesis_signatures.json -genesis=./genesis
//
func main() {
	chainIDPtr := flag.String("chainID", "local_chain", "the ID of the chain")
	erc20SnapshotJSONFilePathPtr := flag.String("erc20snapshot", "./theta_erc20_snapshot.json", "the json file contain the ERC20 balance snapshot")
	stakeDepositFilePathPtr := flag.String("stake_deposit", "./stake_deposit.json", "the initial stake deposits")
	genesisSnapshotFilePathPtr := flag.String("genesis", "./genesis", "the genesis snapshot")
	genesisSignaturesFilePathPtr := flag.String("genesis_signatures", "./genesis_signatures.json", "the genesis signatures")
	flag.Parse()

	chainID := *chainIDPtr
	erc20SnapshotJSONFilePath := *erc20SnapshotJSONFilePathPtr
	stakeDepositFilePath := *stakeDepositFilePathPtr
	genesisSnapshotFilePath := *genesisSnapshotFilePathPtr
	genesisSignaturesFilePath := *genesisSignaturesFilePathPtr

	writeGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath, genesisSignaturesFilePath, genesisSnapshotFilePath)
}

type StakeDeposit struct {
	Source string `json:"source"`
	Holder string `json:"holder"`
	Amount string `json:"amount"`
}

// writeGenesisSnapshot writes genesis snapshot to file system.
func writeGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, stakeDepositFPath, genesisSignaturesFilePath, genesisSnapshotFilePath string) error {
	metadata, sv, err := generateGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, stakeDepositFPath, genesisSignaturesFilePath)
	if err != nil {
		return err
	}
	file, err := os.Create(genesisSnapshotFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = writeMetadata(writer, metadata)
	if err != nil {
		return err
	}

	sv.IncrementHeight()
	writeStoreView(sv, true, writer)
	return err
}

// generateGenesisSnapshot generates the genesis snapshot.
func generateGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, stakeDepositFilePath, genesisSignaturesFilePath string) (*core.SnapshotMetadata, *state.StoreView, error) {
	genesis := &core.SnapshotMetadata{}

	initTFuelToThetaRatio := new(big.Int).SetUint64(5)
	sv := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())

	// --------------- Load initial balances --------------- //

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

		//logger.Infof("address: %v, theta: %v, tfuel: %v", strings.ToLower(address.String()), theta, tfuel))
	}

	// --------------- Perform initial stake deposit --------------- //

	var stakeDeposits []StakeDeposit
	stakeDepositFile, err := os.Open(stakeDepositFilePath)
	stakeDepositByteValue, err := ioutil.ReadAll(stakeDepositFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
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

	genesisHeight := uint64(0)
	hl := &types.HeightList{}
	hl.Append(genesisHeight)
	sv.UpdateStakeTransactionHeightList(hl)

	stateHash := sv.Hash()

	firstBlock := core.NewBlock()
	firstBlock.ChainID = chainID
	firstBlock.Height = genesisHeight
	firstBlock.Epoch = 0
	firstBlock.Parent = common.Hash{}
	firstBlock.StateHash = stateHash
	firstBlock.Timestamp = big.NewInt(time.Now().Unix())

	secondBlock := core.NewBlock()
	secondBlock.ChainID = chainID
	secondBlock.Height = genesisHeight + 1
	secondBlock.Epoch = 0
	secondBlock.Parent = firstBlock.Hash()
	secondBlock.HCC = core.CommitCertificate{BlockHash: firstBlock.Hash()}
	secondBlock.StateHash = stateHash
	secondBlock.Timestamp = big.NewInt(time.Now().Unix())

	thirdBlock := core.NewBlock()
	thirdBlock.Parent = secondBlock.Hash()
	thirdBlock.HCC = core.CommitCertificate{BlockHash: secondBlock.Hash()}

	// --------------- Load genesis signatures --------------- //

	genesisSignaturesFile, err := os.Open(genesisSignaturesFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the genesis signatures file: %v", err))
	}
	defer genesisSignaturesFile.Close()

	var genesisSignatureMap map[string][]byte
	genesisSignaturesMapByteValue, err := ioutil.ReadAll(genesisSignaturesFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal(genesisSignaturesMapByteValue, &genesisSignatureMap)

	validators := consensus.SelectTopStakeHoldersAsValidators(vcp).Validators()
	votes := []core.Vote{}
	for _, validator := range validators {
		vote := core.Vote{
			Block:  secondBlock.Hash(),
			Height: secondBlock.Height,
			ID:     validator.Address,
			Epoch:  secondBlock.Height,
		}
		sigBytes := genesisSignatureMap[validator.Address.String()]
		sig, err := crypto.SignatureFromBytes(sigBytes)
		if err != nil {
			panic(fmt.Sprintf("failed to get genesis signature: %v", err))
		}
		vote.SetSignature(sig)
		votes = append(votes, vote)
	}

	vcpProof, err := proofVCP(sv, state.ValidatorCandidatePoolKey())
	if err != nil {
		panic(fmt.Errorf("Failed to get VCP Proof"))
	}
	genesis.BlockTrios = append(genesis.BlockTrios, core.SnapshotBlockTrio{First: core.SnapshotFirstBlock{Header: *firstBlock.BlockHeader, Proof: *vcpProof}, Second: *secondBlock.BlockHeader, Third: core.SnapshotThirdBlock{Header: *thirdBlock.BlockHeader, Votes: votes}})

	// --------------- Sanity Checks --------------- //

	err = sanityChecks(sv)
	if err != nil {
		panic(fmt.Sprintf("Sanity checks failed: %v", err))
	}

	return genesis, sv, nil
}

func proofVCP(sv *state.StoreView, vcpKey []byte) (*core.VCPProof, error) {
	vp := &core.VCPProof{}
	err := sv.ProveVCP(vcpKey, vp)
	if err != nil {
		return nil, err
	}
	return vp, nil
}

func writeMetadata(writer *bufio.Writer, metadata *core.SnapshotMetadata) error {
	raw, err := rlp.EncodeToBytes(*metadata)
	if err != nil {
		log.Error("Failed to encode snapshot metadata")
		return err
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write snapshot metadata length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write snapshot metadata")
		return err
	}

	meta := &core.SnapshotMetadata{}
	rlp.DecodeBytes(raw, meta)

	return nil
}

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer) {
	height := itobs(sv.Height())
	err := writeRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = writeRecord(writer, k, v)
		if err != nil {
			panic(err)
		}
		return true
	})
	err = writeRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func writeRecord(writer *bufio.Writer, k, v common.Bytes) error {
	record := core.SnapshotTrieRecord{K: k, V: v}
	raw, err := rlp.EncodeToBytes(record)
	if err != nil {
		return fmt.Errorf("Failed to encode storage record, %v", err)
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		return fmt.Errorf("Failed to write storage record length, %v", err)
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		return fmt.Errorf("Failed to write storage record, %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush storage record, %v", err)
	}
	return nil
}

func itobs(val uint64) []byte {
	arr := make([]byte, 8)
	binary.LittleEndian.PutUint64(arr, val)
	return arr
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
