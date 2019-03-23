package snapshot

import (
	"fmt"
	"io"
	"os"

	"github.com/thetatoken/theta/core"
)

func ImportChainBackup(filePath string) (*core.BackupBlock, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var block *core.BackupBlock
	for {
		backupBlock := &core.BackupBlock{}
		err := core.ReadRecord(file, backupBlock)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Failed to read backup record, %v", err)
		}

		block = backupBlock
	}

	if block == nil {
		return nil, fmt.Errorf("Failed to find any block from backup")
	}
	return block, nil
}
