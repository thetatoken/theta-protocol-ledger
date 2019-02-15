package rpc

type PruneArgs struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type PruneResult struct {
}

func (t *ThetaRPCService) ExecutePrune(args *PruneArgs, result *PruneResult) error {
	start := args.Start
	end := args.End
	err := t.ledger.PruneState(start, end)
	return err
}
