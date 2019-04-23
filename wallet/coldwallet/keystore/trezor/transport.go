package trezor

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thetatoken/theta/common"
)

const TREZORD_HOST = "http://127.0.0.1:21325"

const OUTDATED_FIRMWARE_ERROR = `Your Trezor firmware is out of date. Update it with the following command:
trezorctl firmware-update
Or visit https://wallet.trezor.io/`

var TREZORD_VERSION_MODERN = [3]uint32{2, 0, 25}

var MINIMUM_FIRMWARE_VERSION = map[string][3]uint32{
	"1": [3]uint32{1, 8, 0},
	"T": [3]uint32{2, 1, 0},
}

type BridgeTransport struct {
	Device   Device
	Session  string
	Features *Features
	Legacy   bool
	Debug    bool
}

type Session struct {
	Session string
}

type Config struct {
	Version string
}

type Device struct {
	Path         string
	Vendor       int
	Product      int
	Debug        bool
	Session      Session
	DebugSession Session
}

func (b *BridgeTransport) CallRaw(data []byte) (interface{}, MessageType, error) {
	err := b.write("post", data, nil)
	if err != nil {
		return nil, 0, err
	}

	return b.read("read", nil)
}

func (b *BridgeTransport) WriteRaw(data []byte) error {
	return b.write("post", data, nil)
}

func (b *BridgeTransport) ReadRaw() (interface{}, MessageType, error) {
	return b.read("read", nil)
}

func (b *BridgeTransport) isOutdated(version [3]uint32) bool {
	if b.Features.BootloaderMode {
		return false
	}

	var requiredVersion [3]uint32
	if b.Device.Product == 0x53c0 {
		requiredVersion = MINIMUM_FIRMWARE_VERSION["1"]
	} else if b.Device.Product == 0x53c1 {
		requiredVersion = MINIMUM_FIRMWARE_VERSION["1"] //TODO: should be Model T
	}
	return isTupleLT(version, requiredVersion)
}

func (b *BridgeTransport) CheckFirmwareVersion(version [3]uint32, warnOnly bool) error {
	if b.isOutdated(version) {
		if warnOnly {
			fmt.Println(OUTDATED_FIRMWARE_ERROR)
		} else {
			return fmt.Errorf(OUTDATED_FIRMWARE_ERROR)
		}
	}
	return nil
}

func isLegacy() bool {
	config := Config{}
	callBridge("configure", "", &config)

	strs := strings.Split(config.Version, ".")
	t0, _ := strconv.Atoi(strs[0])
	t1, _ := strconv.Atoi(strs[1])
	t2, _ := strconv.Atoi(strs[2])
	tuple := [3]uint32{uint32(t0), uint32(t1), uint32(t2)}
	return isTupleLT(tuple, TREZORD_VERSION_MODERN)
}

func isTupleLT(tuple1, tuple2 [3]uint32) bool {
	if tuple1[0] < tuple2[0] {
		return true
	} else if tuple1[0] == tuple2[0] {
		if tuple1[1] < tuple2[1] {
			return true
		} else if tuple1[1] == tuple2[1] {
			if tuple1[2] < tuple2[2] {
				return true
			}
		}
	}
	return false
}

func enumerate() ([]*BridgeTransport, error) {
	isLegacy := isLegacy()
	devices := []Device{}
	_, err := callBridge("enumerate", "", &devices)
	if err != nil {
		return nil, err
	}

	bridgeTransports := []*BridgeTransport{}
	for _, dev := range devices {
		bridgeTransports = append(bridgeTransports, &BridgeTransport{Device: dev, Legacy: isLegacy})
	}
	return bridgeTransports, nil
}

func (b *BridgeTransport) BeginSession() error {
	if b.Device.Path == "" {
		transports, err := enumerate()
		if err != nil {
			return err
		}
		if len(transports) > 0 {
			transport := transports[0]
			b.Device = transport.Device
		} else {
			return fmt.Errorf("Can't find any deviced attached")
		}
	}

	session := Session{}
	err := b.write("acquire/"+b.Device.Path, nil, &session)
	if err != nil {
		return err
	}
	b.Session = session.Session
	return nil
}

func (b *BridgeTransport) EndSession() {
	if b.Session == "" {
		return
	}
	b.write("release", nil, nil)
	b.Session = ""
}

func callBridge(uri, dataStr string, target interface{}) ([]byte, error) {
	url := TREZORD_HOST + "/" + uri
	req, err := http.NewRequest("POST", url, strings.NewReader(dataStr))
	req.Header.Set("Origin", "https://python.trezor.io")
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("trezord: %v failed with code %v: %v, %v", uri, resp.StatusCode, resp.Proto, resp.Body)
	}

	if target != nil {
		return nil, json.NewDecoder(resp.Body).Decode(target)
	}
	return ioutil.ReadAll(resp.Body)
}

func (b *BridgeTransport) write(action string, data []byte, target interface{}) error {
	var session string
	if b.Session == "" {
		session = "null"
	} else {
		session = b.Session
	}
	uri := action + "/" + session
	_, err := callBridge(uri, common.Bytes2Hex(data), target)
	return err
}

func Pack(packed *[6]byte, msgType uint16, serLen uint32) {
	binary.BigEndian.PutUint16(packed[:2], msgType)
	binary.BigEndian.PutUint32(packed[2:], serLen)
}

func unpack(packed [6]byte) (msgType uint16, serLen uint32) {
	msgType = uint16(binary.BigEndian.Uint16(packed[:2]))
	serLen = binary.BigEndian.Uint32(packed[2:])
	return
}

func (b *BridgeTransport) read(action string, data []byte) (interface{}, MessageType, error) {
	var session string
	if b.Session == "" {
		session = "null"
	} else {
		session = b.Session
	}
	uri := action + "/" + session
	respData, err := callBridge(uri, common.Bytes2Hex(data), nil)
	if err != nil {
		return nil, 0, err
	}

	respData = ConvertBytes(respData)
	header := [6]byte{}
	copy(header[:], respData[:6])

	msgType, _ := unpack(header)
	maxLimit := int32(1<<31 - 1)
	target := GetEmptyObj(MessageType(msgType))
	resp, err := LoadMessage(strings.NewReader(string(respData[6:])), target, &maxLimit)
	return resp, MessageType(msgType), err
}

func ConvertBytes(bytes []byte) (res []byte) {
	for i := range bytes {
		if i%2 == 0 {
			continue
		}
		res = append(res, convertBytePair([]byte{bytes[i-1], bytes[i]}))
	}
	return
}

func convertBytePair(pair []byte) byte {
	s := string(pair)
	b, _ := strconv.ParseUint(s, 16, 0)
	return byte(b)
}

func RevertBytes(bytes []byte) (res []byte) {
	for i := range bytes {
		res = append(res, revertBytePair(bytes[i])...)
	}
	return
}

func revertBytePair(b byte) []byte {
	hex := common.Bytes2Hex([]byte{b})
	return []byte{hex[0], hex[1]}
}
