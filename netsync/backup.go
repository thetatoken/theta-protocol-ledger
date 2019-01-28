package netsync

import (
	"fmt"
	"io"
	"os"

	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
)

func LoadBackup(filePath string, db database.Database) (*core.BlockHeader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	kvstore := kvstore.NewKVStore(db)

	var backupBlock *core.BackupBlock
	for {
		backupBlock = &core.BackupBlock{}
		err := core.ReadRecord(file, backupBlock)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Failed to read backup record, %v", err)
		}
		hash := backupBlock.Block.Hash()
		kvstore.Put(hash[:], *backupBlock.Block)
		// TODO: add votes
	}
	if backupBlock != nil {
		return backupBlock.Block.BlockHeader, nil
	}
	return nil, nil
}
