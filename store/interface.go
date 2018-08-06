package store

import "github.com/thetatoken/ukulele/types"

// Store is the interface for key/value storages.
type Store interface {
	Put(key types.Bytes, value interface{}) error
	Delete(key types.Bytes) error
	Get(key types.Bytes) (value interface{}, err error)
}
