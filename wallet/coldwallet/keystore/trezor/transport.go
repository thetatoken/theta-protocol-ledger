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

// // USB vendor/product IDs for Trezors
// DEV_TREZOR1 = (0x534C, 0x0001)
// DEV_TREZOR2 = (0x1209, 0x53C1)
// DEV_TREZOR2_BL = (0x1209, 0x53C0)

// TREZORS = {DEV_TREZOR1, DEV_TREZOR2, DEV_TREZOR2_BL}

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
	Session string `json:"session"`
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
	fmt.Printf("############# CALLING RAW ########### data: %v\n", data)
	err := b.write("post", data, nil)
	if err != nil {
		return nil, 0, err
	}

	return b.read("read", nil)
}

func (b *BridgeTransport) CallRawWrite(data []byte) error {
	return b.write("post", data, nil)
}

func (b *BridgeTransport) CallRawRead() (interface{}, MessageType, error) {
	return b.read("read", nil)
}

func (b *BridgeTransport) isOutdated(version [3]uint32) bool {
	if b.Features.BootloaderMode {
		return false
	}

	// var requiredVersion [3]uint32
	// if b.Device.Product == 0x53c0 { // model 1
	// 	requiredVersion = MINIMUM_FIRMWARE_VERSION["1"]
	// } else if b.Device.Product == 0x53c1 { // model T
	// 	requiredVersion = MINIMUM_FIRMWARE_VERSION["T"]
	// }
	// return isTupleLT(version, requiredVersion)
	return false
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
	// req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))

	fmt.Printf("<<<<<<<<<<<<<<<< CALL BRIDGE >>>>>>>>>>>>>> %v, %v\n", url, dataStr)

	req, err := http.NewRequest("POST", url, strings.NewReader(dataStr))
	req.Header.Set("Origin", "https://python.trezor.io")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("~~~~~~~~~~~~~~write~~~~~~~~~~~~~~~calling bridge: %v failed with code %v: %v, %v", uri, resp.StatusCode, resp.Proto, resp.Body)
		return nil, fmt.Errorf("trezord: %v failed with code %v: %v, %v", uri, resp.StatusCode, resp.Proto, resp.Body)
	}

	fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~Resp -> status: %v, body: %v\n", resp.Status, resp.Body)
	if target != nil {
		return nil, json.NewDecoder(resp.Body).Decode(target)
	} else {
		return ioutil.ReadAll(resp.Body)
	}
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
	fmt.Printf("----------------->>> msgType: %v\n", int32(msgType))
	fmt.Printf("----------------->>> data: %v\n", respData[6:])
	fmt.Printf("----------------->>> str: %v\n", string(respData[6:]))
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

// func callBridgeRead(uri, dataStr string) ([]byte, error) {
// 	url := TREZORD_HOST + "/" + uri
// 	// req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
// 	req, err := http.NewRequest("POST", url, strings.NewReader(dataStr))
// 	req.Header.Set("Origin", "https://python.trezor.io")

// 	client := &http.Client{Timeout: time.Second * 10}
// 	fmt.Printf("<<<<<<<<<<<<<<<< CALL BRIDGE >>>>>>>>>>>>>> url: %v\n", url)
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, nil
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		fmt.Printf("~~~~~~~~~~~~~~read~~~~~~~~~~~~~~~calling bridge: %v failed with code %v: %v, %v", uri, resp.StatusCode, resp.Proto, resp.Body)
// 		return nil, fmt.Errorf("trezord: %v failed with code %v: %v, %v", uri, resp.StatusCode, resp.Proto, resp.Body)
// 	}
// 	fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~Resp -> status: %v, body: %v\n", resp.Status, resp.Body)
// 	return ioutil.ReadAll(resp.Body)
// }
