package rpc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/crypto/bls"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/mempool"
	"github.com/thetatoken/theta/version"
)

// ------------------------------- GetVersion -----------------------------------

type GetVersionArgs struct {
}

type GetVersionResult struct {
	Version   string `json:"version"`
	GitHash   string `json:"git_hash"`
	Timestamp string `json:"timestamp"`
}

func (t *ThetaRPCService) GetVersion(args *GetVersionArgs, result *GetVersionResult) (err error) {
	result.Version = version.Version
	result.GitHash = version.GitHash
	result.Timestamp = version.Timestamp
	return nil
}

// ------------------------------- GetAccount -----------------------------------

type GetAccountArgs struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Preview bool   `json:"preview"` // preview the account balance from the ScreenedView
}

type GetAccountResult struct {
	*types.Account
	Address string `json:"address"`
}

func (t *ThetaRPCService) GetAccount(args *GetAccountArgs, result *GetAccountResult) (err error) {
	if args.Address == "" {
		return errors.New("Address must be specified")
	}
	address := common.HexToAddress(args.Address)
	result.Address = args.Address

	var ledgerState *state.StoreView
	if args.Preview {
		ledgerState, err = t.ledger.GetScreenedSnapshot()
	} else {
		ledgerState, err = t.ledger.GetFinalizedSnapshot()
	}
	if err != nil {
		return err
	}

	account := ledgerState.GetAccount(address)
	if account == nil {
		return fmt.Errorf("Account with address %s is not found", address.Hex())
	}
	account.UpdateToHeight(ledgerState.Height())

	result.Account = account
	return nil
}

// ------------------------------- GetSplitRule -----------------------------------

type GetSplitRuleArgs struct {
	ResourceID string `json:"resource_id"`
}

type GetSplitRuleResult struct {
	*types.SplitRule
}

func (t *ThetaRPCService) GetSplitRule(args *GetSplitRuleArgs, result *GetSplitRuleResult) (err error) {
	if args.ResourceID == "" {
		return errors.New("ResourceID must be specified")
	}
	resourceID := args.ResourceID
	ledgerState, err := t.ledger.GetDeliveredSnapshot()
	if err != nil {
		return err
	}
	result.SplitRule = ledgerState.GetSplitRule(resourceID)
	return nil
}

// ------------------------------ GetTransaction -----------------------------------

type GetTransactionArgs struct {
	Hash string `json:"hash"`
}

type GetTransactionResult struct {
	BlockHash   common.Hash                `json:"block_hash"`
	BlockHeight common.JSONUint64          `json:"block_height"`
	Status      TxStatus                   `json:"status"`
	TxHash      common.Hash                `json:"hash"`
	Type        byte                       `json:"type"`
	Tx          types.Tx                   `json:"transaction"`
	Receipt     *blockchain.TxReceiptEntry `json:"receipt"`
}

type TxStatus string

const (
	TxStatusNotFound  = "not_found"
	TxStatusPending   = "pending"
	TxStatusFinalized = "finalized"
	TxStatusAbandoned = "abandoned"
)

func (t *ThetaRPCService) GetTransaction(args *GetTransactionArgs, result *GetTransactionResult) (err error) {
	if args.Hash == "" {
		return errors.New("Transanction hash must be specified")
	}
	hash := common.HexToHash(args.Hash)
	result.TxHash = hash

	raw, block, found := t.chain.FindTxByHash(hash)
	if !found {
		txStatus, exists := t.mempool.GetTransactionStatus(args.Hash)
		if exists {
			if txStatus == mempool.TxStatusAbandoned {
				result.Status = TxStatusAbandoned
			} else {
				result.Status = TxStatusPending
			}
		} else {
			result.Status = TxStatusNotFound
		}
		return nil
	}
	result.BlockHash = block.Hash()
	result.BlockHeight = common.JSONUint64(block.Height)

	if block.Status.IsFinalized() {
		result.Status = TxStatusFinalized
	} else {
		result.Status = TxStatusPending
	}

	tx, err := types.TxFromBytes(raw)
	if err != nil {
		return err
	}
	result.Tx = tx
	result.Type = getTxType(tx)

	// Add receipt
	receipt, found := t.chain.FindTxReceiptByHash(hash)
	if found {
		result.Receipt = receipt
	}

	return nil
}

// ------------------------------ GetPendingTransactions -----------------------------------

type GetPendingTransactionsArgs struct {
}

type GetPendingTransactionsResult struct {
	TxHashes []string `json:"tx_hashes"`
}

func (t *ThetaRPCService) GetPendingTransactions(args *GetPendingTransactionsArgs, result *GetPendingTransactionsResult) (err error) {
	pendingTxHashes := t.mempool.GetCandidateTransactionHashes()
	result.TxHashes = pendingTxHashes
	return nil
}

// ------------------------------ GetBlock -----------------------------------

type GetBlockArgs struct {
	Hash common.Hash `json:"hash"`
}

type Tx struct {
	types.Tx `json:"raw"`
	Type     byte                       `json:"type"`
	Hash     common.Hash                `json:"hash"`
	Receipt  *blockchain.TxReceiptEntry `json:"receipt"`
}

type GetBlockResult struct {
	*GetBlockResultInner
}

type GetBlocksResult []*GetBlockResultInner

type GetBlockResultInner struct {
	ChainID       string                 `json:"chain_id"`
	Epoch         common.JSONUint64      `json:"epoch"`
	Height        common.JSONUint64      `json:"height"`
	Parent        common.Hash            `json:"parent"`
	TxHash        common.Hash            `json:"transactions_hash"`
	StateHash     common.Hash            `json:"state_hash"`
	Timestamp     *common.JSONBig        `json:"timestamp"`
	Proposer      common.Address         `json:"proposer"`
	HCC           core.CommitCertificate `json:"hcc"`
	GuardianVotes *core.AggregatedVotes  `json:"guardian_votes"`

	Children []common.Hash    `json:"children"`
	Status   core.BlockStatus `json:"status"`

	Hash common.Hash `json:"hash"`
	Txs  []Tx        `json:"transactions"`
}

type TxType byte

const (
	TxTypeCoinbase = byte(iota)
	TxTypeSlash
	TxTypeSend
	TxTypeReserveFund
	TxTypeReleaseFund
	TxTypeServicePayment
	TxTypeSplitRule
	TxTypeSmartContract
	TxTypeDepositStake
	TxTypeWithdrawStake
	TxTypeDepositStakeTxV2
)

func (t *ThetaRPCService) GetBlock(args *GetBlockArgs, result *GetBlockResult) (err error) {
	if args.Hash.IsEmpty() {
		return errors.New("Block hash must be specified")
	}

	block, err := t.chain.FindBlock(args.Hash)
	if err != nil {
		return err
	}

	result.GetBlockResultInner = &GetBlockResultInner{}
	result.ChainID = block.ChainID
	result.Epoch = common.JSONUint64(block.Epoch)
	result.Height = common.JSONUint64(block.Height)
	result.Parent = block.Parent
	result.TxHash = block.TxHash
	result.StateHash = block.StateHash
	result.Timestamp = (*common.JSONBig)(block.Timestamp)
	result.Proposer = block.Proposer
	result.Children = block.Children
	result.Status = block.Status
	result.HCC = block.HCC
	result.GuardianVotes = block.GuardianVotes

	result.Hash = block.Hash()

	// Parse and fulfill Txs.
	var tx types.Tx
	for _, txBytes := range block.Txs {
		tx, err = types.TxFromBytes(txBytes)
		if err != nil {
			return
		}
		hash := crypto.Keccak256Hash(txBytes)

		tp := getTxType(tx)
		txw := Tx{
			Tx:   tx,
			Hash: hash,
			Type: tp,
		}

		receipt, found := t.chain.FindTxReceiptByHash(hash)
		if found {
			txw.Receipt = receipt
		}

		result.Txs = append(result.Txs, txw)
	}
	return
}

// ------------------------------ GetBlockByHeight -----------------------------------

type GetBlockByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

func (t *ThetaRPCService) GetBlockByHeight(args *GetBlockByHeightArgs, result *GetBlockResult) (err error) {
	// if args.Height == 0 {
	// 	return errors.New("Block height must be specified")
	// }

	blocks := t.chain.FindBlocksByHeight(uint64(args.Height))

	var block *core.ExtendedBlock
	for _, b := range blocks {
		if b.Status.IsFinalized() {
			block = b
			break
		}
	}

	if block == nil {
		return
	}

	result.GetBlockResultInner = &GetBlockResultInner{}
	result.ChainID = block.ChainID
	result.Epoch = common.JSONUint64(block.Epoch)
	result.Height = common.JSONUint64(block.Height)
	result.Parent = block.Parent
	result.TxHash = block.TxHash
	result.StateHash = block.StateHash
	result.Timestamp = (*common.JSONBig)(block.Timestamp)
	result.Proposer = block.Proposer
	result.Children = block.Children
	result.Status = block.Status
	result.HCC = block.HCC
	result.GuardianVotes = block.GuardianVotes

	result.Hash = block.Hash()

	// Parse and fulfill Txs.
	var tx types.Tx
	for _, txBytes := range block.Txs {
		tx, err = types.TxFromBytes(txBytes)
		if err != nil {
			return
		}
		hash := crypto.Keccak256Hash(txBytes)

		tp := getTxType(tx)
		txw := Tx{
			Tx:   tx,
			Hash: hash,
			Type: tp,
		}

		receipt, found := t.chain.FindTxReceiptByHash(hash)
		if found {
			txw.Receipt = receipt
		}

		result.Txs = append(result.Txs, txw)
	}
	return
}

// ------------------------------ GetBlocksByRange -----------------------------------

type GetBlocksByRangeArgs struct {
	Start common.JSONUint64 `json:"start"`
	End common.JSONUint64 `json:"end"`
}

func (t *ThetaRPCService) GetBlocksByRange(args *GetBlocksByRangeArgs, result *GetBlocksResult) (err error) {
	if args.Start == 0 && args.End == 0 {
		return errors.New("Starting block and ending block must be specified")
	}

	if args.Start > args.End {
		return errors.New("Starting block must be less than ending block")
	}

	if args.End - args.Start > 100  {
		return errors.New("Can't retrieve more than 100 blocks at a time")
	}

	blocks := t.chain.FindBlocksByHeight(uint64(args.End))

	var block *core.ExtendedBlock
	for _, b := range blocks {
		if b.Status.IsFinalized() {
			block = b
			break
		}
	}

	if block == nil {
		return
	}

	for common.JSONUint64(block.Height) >= args.Start {
		blkInner := &GetBlockResultInner{}
		blkInner.ChainID = block.ChainID
		blkInner.Epoch = common.JSONUint64(block.Epoch)
		blkInner.Height = common.JSONUint64(block.Height)
		blkInner.Parent = block.Parent
		blkInner.TxHash = block.TxHash
		blkInner.StateHash = block.StateHash
		blkInner.Timestamp = (*common.JSONBig)(block.Timestamp)
		blkInner.Proposer = block.Proposer
		blkInner.Children = block.Children
		blkInner.Status = block.Status
		blkInner.HCC = block.HCC
		blkInner.GuardianVotes = block.GuardianVotes

		blkInner.Hash = block.Hash()

		// Parse and fulfill Txs.
		var tx types.Tx
		for _, txBytes := range block.Txs {
			tx, err = types.TxFromBytes(txBytes)
			if err != nil {
				return
			}
			hash := crypto.Keccak256Hash(txBytes)

			t := getTxType(tx)
			txw := Tx{
				Tx:   tx,
				Hash: hash,
				Type: t,
			}
			blkInner.Txs = append(blkInner.Txs, txw)
		}

		*result = append([]*GetBlockResultInner{blkInner}, *result...)

		block, err = t.chain.FindBlock(block.Parent)
		if err != nil {
			return err
		}
	}
	return
}

// ------------------------------ GetStatus -----------------------------------

type GetStatusArgs struct{}

type GetStatusResult struct {
	Address                    string            `json:"address"`
	ChainID                    string            `json:"chain_id"`
	PeerID                     string            `json:"peer_id"`
	LatestFinalizedBlockHash   common.Hash       `json:"latest_finalized_block_hash"`
	LatestFinalizedBlockHeight common.JSONUint64 `json:"latest_finalized_block_height"`
	LatestFinalizedBlockTime   *common.JSONBig   `json:"latest_finalized_block_time"`
	LatestFinalizedBlockEpoch  common.JSONUint64 `json:"latest_finalized_block_epoch"`
	CurrentEpoch               common.JSONUint64 `json:"current_epoch"`
	CurrentHeight              common.JSONUint64 `json:"current_height"`
	CurrentTime                *common.JSONBig   `json:"current_time"`
	Syncing                    bool              `json:"syncing"`
}

func (t *ThetaRPCService) GetStatus(args *GetStatusArgs, result *GetStatusResult) (err error) {
	s := t.consensus.GetSummary()
	result.Address = t.consensus.ID()
	//result.PeerID = t.dispatcher.ID()
	result.PeerID = t.dispatcher.LibP2PID() // TODO: use ID() instead after 1.3.0 upgrade
	result.ChainID = t.consensus.Chain().ChainID
	latestFinalizedHash := s.LastFinalizedBlock
	if !latestFinalizedHash.IsEmpty() {
		result.LatestFinalizedBlockHash = latestFinalizedHash
		block, err := t.chain.FindBlock(latestFinalizedHash)
		if err != nil {
			return err
		}
		result.LatestFinalizedBlockEpoch = common.JSONUint64(block.Epoch)
		result.LatestFinalizedBlockHeight = common.JSONUint64(block.Height)
		result.LatestFinalizedBlockTime = (*common.JSONBig)(block.Timestamp)
		result.Syncing = isSyncing(block)
	}
	result.CurrentEpoch = common.JSONUint64(s.Epoch)
	result.CurrentTime = (*common.JSONBig)(big.NewInt(time.Now().Unix()))

	maxVoteHeight := uint64(0)
	epochVotes, err := t.consensus.State().GetEpochVotes()
	if err != nil {
		return err
	}
	if epochVotes != nil {
		for _, v := range epochVotes.Votes() {
			if v.Height > maxVoteHeight {
				maxVoteHeight = v.Height
			}
		}
		result.CurrentHeight = common.JSONUint64(maxVoteHeight - 1) // current finalized height is at most maxVoteHeight-1
	}
	return
}

// ------------------------------ GetPeers -----------------------------------

type GetPeersArgs struct{}

type GetPeersResult struct {
	Peers []string `json:"peers"`
}

func (t *ThetaRPCService) GetPeers(args *GetPeersArgs, result *GetPeersResult) (err error) {
	peers := t.dispatcher.Peers()
	result.Peers = peers

	return
}

// ------------------------------ GetVcp -----------------------------------

type GetVcpByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

type GetVcpResult struct {
	BlockHashVcpPairs []BlockHashVcpPair
}

type BlockHashVcpPair struct {
	BlockHash  common.Hash
	Vcp        *core.ValidatorCandidatePool
	HeightList *types.HeightList
}

func (t *ThetaRPCService) GetVcpByHeight(args *GetVcpByHeightArgs, result *GetVcpResult) (err error) {
	deliveredView, err := t.ledger.GetDeliveredSnapshot()
	if err != nil {
		return err
	}

	db := deliveredView.GetDB()
	height := uint64(args.Height)

	blockHashVcpPairs := []BlockHashVcpPair{}
	blocks := t.chain.FindBlocksByHeight(height)
	for _, b := range blocks {
		blockHash := b.Hash()
		stateRoot := b.StateHash
		blockStoreView := state.NewStoreView(height, stateRoot, db)
		if blockStoreView == nil { // might have been pruned
			return fmt.Errorf("the VCP for height %v does not exists, it might have been pruned", height)
		}
		vcp := blockStoreView.GetValidatorCandidatePool()
		hl := blockStoreView.GetStakeTransactionHeightList()
		blockHashVcpPairs = append(blockHashVcpPairs, BlockHashVcpPair{
			BlockHash:  blockHash,
			Vcp:        vcp,
			HeightList: hl,
		})
	}

	result.BlockHashVcpPairs = blockHashVcpPairs

	return nil
}

// ------------------------------ GetGcp -----------------------------------

type GetGcpByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

type GetGcpResult struct {
	BlockHashGcpPairs []BlockHashGcpPair
}

type BlockHashGcpPair struct {
	BlockHash common.Hash
	Gcp       *core.GuardianCandidatePool
}

func (t *ThetaRPCService) GetGcpByHeight(args *GetGcpByHeightArgs, result *GetGcpResult) (err error) {
	deliveredView, err := t.ledger.GetDeliveredSnapshot()
	if err != nil {
		return err
	}

	db := deliveredView.GetDB()
	height := uint64(args.Height)

	blockHashGcpPairs := []BlockHashGcpPair{}
	blocks := t.chain.FindBlocksByHeight(height)
	for _, b := range blocks {
		blockHash := b.Hash()
		stateRoot := b.StateHash
		blockStoreView := state.NewStoreView(height, stateRoot, db)
		if blockStoreView == nil { // might have been pruned
			return fmt.Errorf("the GCP for height %v does not exists, it might have been pruned", height)
		}
		gcp := blockStoreView.GetGuardianCandidatePool()
		blockHashGcpPairs = append(blockHashGcpPairs, BlockHashGcpPair{
			BlockHash: blockHash,
			Gcp:       gcp,
		})
	}

	result.BlockHashGcpPairs = blockHashGcpPairs

	return nil
}

// ------------------------------ GetGuardianKey -----------------------------------

type GetGuardianInfoArgs struct{}

type GetGuardianInfoResult struct {
	BLSPubkey string
	BLSPop    string
	Address   string
	Signature string
}

func (t *ThetaRPCService) GetGuardianInfo(args *GetGuardianInfoArgs, result *GetGuardianInfoResult) (err error) {
	privKey := t.consensus.PrivateKey()
	blsKey, err := bls.GenKey(strings.NewReader(common.Bytes2Hex(privKey.PublicKey().ToBytes())))
	if err != nil {
		return fmt.Errorf("Failed to get BLS key: %v", err.Error())
	}

	result.Address = privKey.PublicKey().Address().Hex()
	result.BLSPubkey = hex.EncodeToString(blsKey.PublicKey().ToBytes())
	popBytes := blsKey.PopProve().ToBytes()
	result.BLSPop = hex.EncodeToString(popBytes)

	sig, err := privKey.Sign(popBytes)
	if err != nil {
		return fmt.Errorf("Failed to generate signature: %v", err.Error())
	}
	result.Signature = hex.EncodeToString(sig.ToBytes())

	return nil
}

// ------------------------------ Utils ------------------------------

func getTxType(tx types.Tx) byte {
	t := byte(0x0)
	switch tx.(type) {
	case *types.CoinbaseTx:
		t = TxTypeCoinbase
	case *types.SlashTx:
		t = TxTypeSlash
	case *types.SendTx:
		t = TxTypeSend
	case *types.ReserveFundTx:
		t = TxTypeReserveFund
	case *types.ReleaseFundTx:
		t = TxTypeReleaseFund
	case *types.ServicePaymentTx:
		t = TxTypeServicePayment
	case *types.SplitRuleTx:
		t = TxTypeSplitRule
	case *types.SmartContractTx:
		t = TxTypeSmartContract
	case *types.DepositStakeTx:
		t = TxTypeDepositStake
	case *types.WithdrawStakeTx:
		t = TxTypeWithdrawStake
	case *types.DepositStakeTxV2:
		t = TxTypeDepositStakeTxV2
	}

	return t
}

func isSyncing(block *core.ExtendedBlock) bool {
	currentTime := big.NewInt(time.Now().Unix())
	maxDiff := new(big.Int).SetUint64(30) // thirty seconds, about 5 blocks
	threshold := new(big.Int).Sub(currentTime, maxDiff)
	isSyncing := block.Timestamp.Cmp(threshold) < 0
	return isSyncing
}
