package metrics

const (
	MHeartBeat = "heartbeat"

	MConsensusFinalized    = "consensus.finalized"
	MConsensusCommitted    = "consensus.committed"
	MConsensusInvalidBlock = "consensus.invalid_block"
	MConsensusValidBlock   = "consensus.valid_block"
	MConsensusValidVote    = "consensus.valid_vote"
	MConsensusInvalidVote  = "consensus.valid_vote"
	MConsensusFinalizedTxs = "consensus.finalized_txs"

	MMempoolTxs = "mempool.txs"

	MSyncPendingHashes = "sync.pending_hashes"
	MSyncOrphanBlocks  = "sync.orphan_blocks"

	MSyncInvRequestReceived   = "sync.inventory_requests_received"
	MSyncInvRequestSent       = "sync.inventory_requests_sent"
	MSyncInvResponseReceived  = "sync.inventory_response_received"
	MSyncInvResponseSent      = "sync.inventory_response_sent"
	MSyncDataRequestReceived  = "sync.data_requests_received"
	MSyncDataRequestSent      = "sync.data_requests_sent"
	MSyncDataResponseReceived = "sync.data_response_received"
	MSyncDataResponseSent     = "sync.data_response_sent"

	MSyncBlockReceived = "sync.block_received"
	MSyncBlockSent     = "sync.block_sent"
	MSyncVoteReceived  = "sync.vote_received"
	MSyncVoteSent      = "sync.vote_sent"

	MRPCBroadcastRawTx      = "rpc.broadcast_raw_tx"
	MRPCBroadcastRawTxAsync = "rpc.broadcast_raw_tx_async"
	MRPCGetVersion          = "rpc.get_version"
	MRPCGetAccount          = "rpc.get_account"
	MRPCGetSplitRule        = "rpc.get_split_rule"
	MRPCGetTx               = "rpc.get_tx"
	MRPCGetPendingTx        = "rpc.get_pending_tx"
	MRPCGetBlock            = "rpc.get_block"
	MRPCGetBlockByHeight    = "rpc.get_block_by_height"
	MRPCGetStatus           = "rpc.get_status"
	MRPCGetVCP              = "rpc.get_vcp"
	MRPCBackupChain         = "rpc.backup_chain"
	MRPCBackup              = "rpc.backup"

	MPerfSigCacheHit      = "perf.sig_cache_hit"
	VPerfSigCacheHit_Hit  = 100
	VPerfSigCacheHit_Miss = 0
)
