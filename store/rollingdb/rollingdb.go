package rollingdb

import (
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/store/database"
)

var logger = util.GetLoggerForModule("rollingdb")

type RollingDB struct {
	mu sync.RWMutex

	parentPath string // path to parent folder

	root  database.Database
	chain *blockchain.Chain

	rootLayer   *DBLayer
	layers      []*DBLayer // all layers excluding root layer and active layer, ordered from old to new.
	activeLayer *DBLayer

	compactC chan struct{}
}

func NewRollingDB(parentPath string, root database.Database) *RollingDB {
	rootLayer := &DBLayer{
		dbPath: path.Join(parentPath, "db"),
		db:     root,
		name:   0,
	}

	rollingPath := path.Join(parentPath, "db", "rolling")
	_ = os.Mkdir(rollingPath, 0700)

	rdb := &RollingDB{
		parentPath: parentPath,
		root:       root,
		rootLayer:  rootLayer,
		compactC:   make(chan struct{}, 1),
	}
	activeLayer, layers := rdb.loadLayers(rollingPath)
	rdb.activeLayer = activeLayer
	rdb.layers = layers

	logger.Debugf("Number of layers after loading DB: %v", len(rdb.layers))
	return rdb

}

func (rdb *RollingDB) SetChain(chain *blockchain.Chain) {
	rdb.chain = chain
}

func (rdb *RollingDB) loadLayers(rollingPath string) (*DBLayer, []*DBLayer) {
	files, err := ioutil.ReadDir(rollingPath)
	if err != nil {
		logger.Panicf("Failed to load layers", err)
	}
	names := []int{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		i, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}
		names = append(names, i)
	}

	if len(names) == 0 {
		if !viper.GetBool(common.CfgStorageRollingEnabled) {
			return rdb.rootLayer, nil
		}
		return NewDBLayer(rollingPath, 1), nil
	}

	sort.Sort(sort.IntSlice(names))
	activeLayer := NewDBLayer(rollingPath, names[len(names)-1])
	layers := []*DBLayer{}
	for _, name := range names[:len(names)-1] {
		layer := NewDBLayer(rollingPath, name)
		layers = append(layers, layer)
	}
	return activeLayer, layers
}

func (rdb *RollingDB) Tag(height uint64, stateRoot common.Hash) {
	if !viper.GetBool(common.CfgStorageRollingEnabled) {
		return
	}

	logger.Debugf("Tag: height=%v, root=%v", height, stateRoot.Hex())
	rdb.activeLayer.addTag(height, stateRoot)

	if isRollingHeight(height) {
		rdb.addLayer()
	}

	if isCompactionHeight(height) {
		go rdb.compact(height)
	}
}

func (rdb *RollingDB) addLayer() {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	rollingPath := path.Join(rdb.parentPath, "db", "rolling")

	rdb.layers = append(rdb.layers, rdb.activeLayer)
	rdb.activeLayer = NewDBLayer(rollingPath, rdb.activeLayer.name+1)

	logger.Debugf("Added new layer: name=%v", rdb.activeLayer.name)
}

func (rdb *RollingDB) compact(height uint64) {
	if !viper.GetBool(common.CfgStorageStatePruningEnabled) {
		return
	}

	select {
	case rdb.compactC <- struct{}{}: // Make sure there is only one active compaction task
		logger.Infof("Starting compaction")

		start := time.Now()
		defer func() {
			logger.Infof("Compaction finished in %v", time.Since(start))
		}()

		defer func() {
			<-rdb.compactC
		}()

		logger.Debugf("Number of layers: %v", len(rdb.layers))
		if len(rdb.layers) == 0 {
			logger.Infof("No rolling DB layer found, skip compaction")
			return
		}

		// Copying state from source to target
		targetLayer := rdb.activeLayer
		var sourceLayer *DBLayer

		// Look for layers to cut off
		minimumNumBlocksToRetain := uint64(viper.GetInt(common.CfgStorageStatePruningRetainedBlocks))
		found := false
		for i := len(rdb.layers) - 1; i >= 0; i-- {
			if height-rdb.layers[i].tag.Height > minimumNumBlocksToRetain+10 {
				found = true
				sourceLayer = rdb.layers[i]
				break
			}
		}
		if !found {
			logger.Info("No layer old enough to cut off")
			return
		}

		if !isRollingHeight(sourceLayer.tag.Height) {
			// potentially db was not cut off cleanly, keep the layer until one cleancut is made
			logger.Infof("Compaction canceled: sourceLayer.name=%v, lastLayer.Height=%v", sourceLayer.name, sourceLayer.tag.Height)
			return
		}

		blocks := rdb.chain.FindBlocksByHeight(sourceLayer.tag.Height)
		logger.Debugf("Found %v blocks for height %v", len(blocks), sourceLayer.tag.Height)

		for _, block := range blocks {
			if block.Status.IsFinalized() {
				logger.Debugf("Found finalized block: %v", block.Hash().Hex())

				for _, stateRoot := range sourceLayer.tag.StateRoots {
					logger.Debugf("State root check, stateRoot: %v, block.StateHash: %v", stateRoot.Hex(), block.StateHash.Hex())

					if stateRoot == block.StateHash {
						logger.Infof("Moving finalized state hash=%v, source=%v, target=%v", stateRoot.Hex(), sourceLayer.name, targetLayer.name)
						copyState(rdb, targetLayer.db.NewBatch(), stateRoot)

						rdb.mu.Lock()
						defer rdb.mu.Unlock()

						remainingLayers := []*DBLayer{}
						for _, layer := range rdb.layers {
							// New layers might have been added after `targetLayer`
							if layer.name <= sourceLayer.name {
								layer.destroy()
							} else {
								remainingLayers = append(remainingLayers, layer)
							}
						}
						rdb.layers = remainingLayers
						break
					}
				}
				break
			}
		}
	default:
		logger.Debugf("Only one active compaction task allowed")
		return
	}

}

func isRollingHeight(height uint64) bool {
	return int(height)%viper.GetInt(common.CfgStorageRollingInterval) == 50
}

func isCompactionHeight(height uint64) bool {
	return int(height)%viper.GetInt(common.CfgStorageRollingInterval) == 70
}

//
// ------ implements database.Database interface -----
//
var _ database.Database = (*RollingDB)(nil)

// Return all layers, ordered from new to old
func (rdb *RollingDB) allLayers() []*DBLayer {
	ret := []*DBLayer{rdb.activeLayer}
	for i := len(rdb.layers) - 1; i >= 0; i-- {
		ret = append(ret, rdb.layers[i])
	}
	ret = append(ret, rdb.rootLayer)
	return ret
}

func (rdb *RollingDB) Get(key []byte) ([]byte, error) {
	rdb.mu.RLock()
	defer rdb.mu.RUnlock()

	var err error
	var result []byte

	for _, layer := range rdb.allLayers() {
		result, err = layer.db.Get(key)
		if err == nil {
			return result, err
		}
	}
	return result, err
}

func (rdb *RollingDB) Has(key []byte) (bool, error) {
	rdb.mu.RLock()
	defer rdb.mu.RUnlock()

	var err error
	var result bool

	for _, layer := range rdb.allLayers() {
		result, err = layer.db.Has(key)
		if err == nil {
			return result, err
		}
	}
	return result, err
}

func (rdb *RollingDB) Put(key []byte, value []byte) error {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	return rdb.activeLayer.db.Put(key, value)
}

func (rdb *RollingDB) Delete(key []byte) error {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	for _, layer := range rdb.allLayers() {
		layer.db.Delete(key)
	}

	return nil
}

func (rdb *RollingDB) Close() {
	for _, dbLayer := range rdb.layers {
		dbLayer.db.Close()
	}
	rdb.activeLayer.db.Close()
	// We leave root db to be closed by outer code
}

type RollingDBBatch struct {
	database.Batch
	rdb *RollingDB
}

func (b *RollingDBBatch) Write() error {
	b.rdb.mu.Lock()
	defer b.rdb.mu.Unlock()

	return b.Batch.Write()
}

func (rdb *RollingDB) NewBatch() database.Batch {
	return &RollingDBBatch{
		Batch: rdb.activeLayer.db.NewBatch(),
		rdb:   rdb,
	}
}

func (rdb *RollingDB) CountReference(key []byte) (int, error) {
	// NOOP
	return 0, nil
}

func (rdb *RollingDB) Reference(key []byte) error {
	// NOOP
	return nil
}

func (rdb *RollingDB) Dereference(key []byte) error {
	// NOOP
	return nil
}
