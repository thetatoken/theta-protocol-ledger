// Adapted for Theta
// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package coldwallet

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/karalabe/hid"

	ks "github.com/thetatoken/ukulele/wallet/coldwallet/keystore"
	"github.com/thetatoken/ukulele/wallet/types"
)

// LedgerScheme is the protocol scheme prefixing account and wallet URLs.
const LedgerScheme = "ledger"

// TrezorScheme is the protocol scheme prefixing account and wallet URLs.
const TrezorScheme = "trezor"

// refreshCycle is the maximum time between wallet refreshes (if USB hotplug
// notifications don't work).
const refreshCycle = time.Second

// refreshThrottling is the minimum time between wallet refreshes to avoid USB
// trashing.
const refreshThrottling = 500 * time.Millisecond

// Hub finds and handles generic USB hardware wallets.
type Hub struct {
	scheme     string           // Protocol scheme prefixing account and wallet URLs.
	vendorID   uint16           // USB vendor identifier used for device discovery
	productIDs []uint16         // USB product identifiers used for device discovery
	usageID    uint16           // USB usage page identifier used for macOS device discovery
	endpointID int              // USB endpoint identifier used for non-macOS device discovery
	makeDriver func() ks.Driver // Factory method to construct a vendor specific driver

	refreshed time.Time      // Time instance when the list of wallets was last refreshed
	wallets   []types.Wallet // List of USB wallet devices currently tracking
	updating  bool           // Whether the event notification loop is running

	quit chan chan error

	stateLock sync.RWMutex // Protects the internals of the hub from racey access

	// TODO(karalabe): remove if hotplug lands on Windows
	commsPend int        // Number of operations blocking enumeration
	commsLock sync.Mutex // Lock protecting the pending counter and enumeration
}

// NewLedgerHub creates a new hardware wallet manager for Ledger devices.
func NewLedgerHub() (*Hub, error) {
	return newHub(LedgerScheme, 0x2c97, []uint16{0x0000 /* Ledger Blue */, 0x0001 /* Ledger Nano S */}, 0xffa0, 0, ks.NewLedgerDriver)
}

// // newTrezorHub creates a new hardware wallet manager for Trezor devices.
// func newTrezorHub() (*Hub, error) {
// 	return newHub(TrezorScheme, 0x534c, []uint16{0x0001 /* Trezor 1 */}, 0xff00, 0, newTrezorDriver)
// }

// newHub creates a new hardware wallet manager for generic USB devices.
func newHub(scheme string, vendorID uint16, productIDs []uint16, usageID uint16, endpointID int, makeDriver func() ks.Driver) (*Hub, error) {
	if !hid.Supported() {
		return nil, errors.New("unsupported platform")
	}
	hub := &Hub{
		scheme:     scheme,
		vendorID:   vendorID,
		productIDs: productIDs,
		usageID:    usageID,
		endpointID: endpointID,
		makeDriver: makeDriver,
		quit:       make(chan chan error),
	}
	hub.refreshWallets()
	return hub, nil
}

// Wallets returns all the currently tracked USB
// devices that appear to be hardware wallets.
func (hub *Hub) Wallets() []types.Wallet {
	// Make sure the list of wallets is up to date
	hub.refreshWallets()

	hub.stateLock.RLock()
	defer hub.stateLock.RUnlock()

	cpy := make([]types.Wallet, len(hub.wallets))
	copy(cpy, hub.wallets)
	return cpy
}

// refreshWallets scans the USB devices attached to the machine and updates the
// list of wallets based on the found devices.
func (hub *Hub) refreshWallets() {
	// Don't scan the USB like crazy it the user fetches wallets in a loop
	hub.stateLock.RLock()
	elapsed := time.Since(hub.refreshed)
	hub.stateLock.RUnlock()

	if elapsed < refreshThrottling {
		return
	}
	// Retrieve the current list of USB wallet devices
	var devices []hid.DeviceInfo

	if runtime.GOOS == "linux" {
		// hidapi on Linux opens the device during enumeration to retrieve some infos,
		// breaking the Ledger protocol if that is waiting for user confirmation. This
		// is a bug acknowledged at Ledger, but it won't be fixed on old devices so we
		// need to prevent concurrent comms ourselves. The more elegant solution would
		// be to ditch enumeration in favor of hotplug events, but that don't work yet
		// on Windows so if we need to hack it anyway, this is more elegant for now.
		hub.commsLock.Lock()
		if hub.commsPend > 0 { // A confirmation is pending, don't refresh
			hub.commsLock.Unlock()
			return
		}
	}
	for _, info := range hid.Enumerate(hub.vendorID, 0) {
		for _, id := range hub.productIDs {
			if info.ProductID == id && (info.UsagePage == hub.usageID || info.Interface == hub.endpointID) {
				devices = append(devices, info)
				break
			}
		}
	}
	if runtime.GOOS == "linux" {
		// See rationale before the enumeration why this is needed and only on Linux.
		hub.commsLock.Unlock()
	}
	// Transform the current list of wallets into the new one
	hub.stateLock.Lock()

	wallets := make([]types.Wallet, 0, len(devices))

	for _, device := range devices {

		walletID := assembleColdWalletID(hub.scheme, device.Path)

		// Drop wallets in front of the next device or those that failed for some reason
		for len(hub.wallets) > 0 {
			// Abort if we're past the current device and found an operational one
			_, failure := hub.wallets[0].Status()
			if compareColdWalletID(hub.wallets[0].ID(), walletID) >= 0 || failure == nil {
				break
			}
			// Drop the stale and failed devices
			hub.wallets = hub.wallets[1:]
		}
		// If there are no more wallets or the device is before the next, wrap new wallet
		if len(hub.wallets) == 0 || compareColdWalletID(hub.wallets[0].ID(), walletID) > 0 {
			wallet, err := NewColdWallet(hub, hub.scheme, device.Path)
			if err != nil {
				panic(fmt.Sprintf("Failed to get wallet: %v", err))
			}

			wallets = append(wallets, wallet)
			continue
		}
		// If the device is the same as the first wallet, keep it
		if compareColdWalletID(hub.wallets[0].ID(), walletID) == 0 {
			wallets = append(wallets, hub.wallets[0])
			hub.wallets = hub.wallets[1:]
			continue
		}
	}
	// Drop any leftover wallets and set the new batch
	hub.refreshed = time.Now()
	hub.wallets = wallets
	hub.stateLock.Unlock()
}
