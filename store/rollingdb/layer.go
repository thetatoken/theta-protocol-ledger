package rollingdb

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database"
)

var layerTagKey = []byte("/layertag")

type layerTag struct {
	Height     uint64
	StateRoots []common.Hash
}

type DBLayer struct {
	dbPath string
	name   int
	tag    *layerTag
	db     database.Database
}

func NewDBLayer(rollingPath string, name int) *DBLayer {
	dbPath := path.Join(rollingPath, fmt.Sprintf("%d", name))
	db, err := NewRawDB(dbPath)
	if err != nil {
		logger.Panicf("Failed to create roll db layer, %v", err)
	}

	l := &DBLayer{
		dbPath: dbPath,
		name:   name,
		db:     db,
	}

	// Load tag
	l.tag = l.loadTag()

	return l
}

func (l *DBLayer) loadTag() *layerTag {
	layerTag := &layerTag{}
	raw, err := l.db.Get(layerTagKey)
	if err == nil {
		err = rlp.DecodeBytes(raw, layerTag)
		if err != nil {
			log.Panicf("Failed to decode layer tag, layer=%v, tag=%v, err=%v", l.name, raw, err)
		}
	}

	l.tag = layerTag

	return layerTag
}

func (l *DBLayer) addTag(height uint64, stateRoot common.Hash) {
	layerTag := l.loadTag()

	if height < layerTag.Height {
		return
	}

	if height == layerTag.Height {
		for _, root := range layerTag.StateRoots {
			if root == stateRoot {
				return
			}
		}
		layerTag.StateRoots = append(layerTag.StateRoots, stateRoot)
	} else {
		// height > layerTag.Height
		layerTag.Height = height
		layerTag.StateRoots = []common.Hash{stateRoot}
	}

	l.tag = layerTag

	raw, err := rlp.EncodeToBytes(layerTag)
	if err != nil {
		log.Panicf("Failed to encode layer tag, layer=%v, tag=%v, err=%v", l.name, raw, err)
	}
	err = l.db.Put(layerTagKey, raw)
	if err != nil {
		log.Panicf("Failed to save layer tag, layer=%v, tag=%v, err=%v", l.name, raw, err)
	}
}

func (l *DBLayer) destroy() {
	l.db.Close()

	os.RemoveAll(l.dbPath)
}
