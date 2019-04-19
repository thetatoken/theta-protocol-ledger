package trezor

import (
	"errors"
	fmt "fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
)

var ErrKeyNotFound = errors.New("KeyNotFound")

func loadUvarint(reader io.Reader) (uint, error) {
	buffer := make([]byte, 1)
	result := uint(0)
	shift := uint(0)
	byt := byte(0x80)
	for {
		res := byt & 0x80
		if res == 0 {
			break
		}
		_, err := reader.Read(buffer)
		if err != nil {
			return 0, err
		}
		byt = buffer[0]
		result += uint(byt&0x7F) << shift
		shift += 7
	}

	return result, nil
}

func dumpUvarint(writer io.Writer, n uint) error {
	buffer := make([]byte, 1)
	var shifted uint
	for {
		shifted = n >> 7
		var m byte
		if shifted > 0 {
			m = 0x80
		}
		buffer[0] = byte(n&0x7F) | m
		_, err := writer.Write(buffer)
		if err != nil {
			return err
		}
		n = shifted
		if shifted == 0 {
			break
		}
	}
	return nil
}

func sint2Uint(n int) uint {
	var m int
	m = n << 1
	if n < 0 {
		m = ^m
	}
	return uint(m)
}

func uint2Sint(n uint) int {
	sign := n & 1
	res := n >> 1
	if sign != 0 {
		res = ^res
	}
	return int(res)
}

type ProtoField struct {
	field   reflect.Value
	ftype   string
	fname   string
	isArray bool
}

func getEmptyObj(msgType MessageType) interface{} {
	tname := MessageType_name[int32(msgType)]
	tname = strings.Split(tname, "_")[1]
	tname = "trezor." + tname

	t := proto.MessageType(tname).Elem()

	switch msgType {
	case 0:
		v := reflect.Indirect(reflect.New(t)).Interface().(Initialize)
		return &v
	case 1:
		v := reflect.Indirect(reflect.New(t)).Interface().(Ping)
		return &v
	case 2:
		v := reflect.Indirect(reflect.New(t)).Interface().(Success)
		return &v
	case 3:
		v := reflect.Indirect(reflect.New(t)).Interface().(Failure)
		return &v
	case 4:
		v := reflect.Indirect(reflect.New(t)).Interface().(ChangePin)
		return &v
	case 5:
		v := reflect.Indirect(reflect.New(t)).Interface().(WipeDevice)
		return &v
	case 6:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareErase)
		return &v
	case 7:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareUpload)
		return &v
	case 8:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareRequest)
		return &v
	case 9:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetEntropy)
		return &v
	case 10:
		v := reflect.Indirect(reflect.New(t)).Interface().(Entropy)
		return &v
	case 11:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetPublicKey)
		return &v
	case 12:
		v := reflect.Indirect(reflect.New(t)).Interface().(PublicKey)
		return &v
	case 13:
		v := reflect.Indirect(reflect.New(t)).Interface().(LoadDevice)
		return &v
	case 14:
		v := reflect.Indirect(reflect.New(t)).Interface().(ResetDevice)
		return &v
	case 15:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignTx)
		return &v
	case 16:
		v := reflect.Indirect(reflect.New(t)).Interface().(SimpleSignTx)
		return &v
	case 17:
		v := reflect.Indirect(reflect.New(t)).Interface().(Features)
		return &v
	case 18:
		v := reflect.Indirect(reflect.New(t)).Interface().(PinMatrixRequest)
		return &v
	case 19:
		v := reflect.Indirect(reflect.New(t)).Interface().(PinMatrixAck)
		return &v
	case 20:
		v := reflect.Indirect(reflect.New(t)).Interface().(Cancel)
		return &v
	case 21:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxRequest)
		return &v
	case 22:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxAck)
		return &v
	case 23:
		v := reflect.Indirect(reflect.New(t)).Interface().(CipherKeyValue)
		return &v
	case 24:
		v := reflect.Indirect(reflect.New(t)).Interface().(ClearSession)
		return &v
	case 25:
		v := reflect.Indirect(reflect.New(t)).Interface().(ApplySettings)
		return &v
	case 26:
		v := reflect.Indirect(reflect.New(t)).Interface().(ButtonRequest)
		return &v
	case 27:
		v := reflect.Indirect(reflect.New(t)).Interface().(ButtonAck)
		return &v
	case 28:
		v := reflect.Indirect(reflect.New(t)).Interface().(ApplyFlags)
		return &v
	case 29:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetAddress)
		return &v
	case 30:
		v := reflect.Indirect(reflect.New(t)).Interface().(Address)
		return &v
	case 32:
		v := reflect.Indirect(reflect.New(t)).Interface().(SelfTest)
		return &v
	case 34:
		v := reflect.Indirect(reflect.New(t)).Interface().(BackupDevice)
		return &v
	case 35:
		v := reflect.Indirect(reflect.New(t)).Interface().(EntropyRequest)
		return &v
	case 36:
		v := reflect.Indirect(reflect.New(t)).Interface().(EntropyAck)
		return &v
	case 38:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignMessage)
		return &v
	case 39:
		v := reflect.Indirect(reflect.New(t)).Interface().(VerifyMessage)
		return &v
	case 40:
		v := reflect.Indirect(reflect.New(t)).Interface().(MessageSignature)
		return &v
	case 41:
		v := reflect.Indirect(reflect.New(t)).Interface().(PassphraseRequest)
		return &v
	case 42:
		v := reflect.Indirect(reflect.New(t)).Interface().(PassphraseAck)
		return &v
	case 43:
		v := reflect.Indirect(reflect.New(t)).Interface().(EstimateTxSize)
		return &v
	case 44:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxSize)
		return &v
	case 45:
		v := reflect.Indirect(reflect.New(t)).Interface().(RecoveryDevice)
		return &v
	case 46:
		v := reflect.Indirect(reflect.New(t)).Interface().(WordRequest)
		return &v
	case 47:
		v := reflect.Indirect(reflect.New(t)).Interface().(WordAck)
		return &v
	case 48:
		v := reflect.Indirect(reflect.New(t)).Interface().(CipheredKeyValue)
		return &v
	case 49:
		v := reflect.Indirect(reflect.New(t)).Interface().(EncryptMessage)
		return &v
	case 50:
		v := reflect.Indirect(reflect.New(t)).Interface().(EncryptedMessage)
		return &v
	case 51:
		v := reflect.Indirect(reflect.New(t)).Interface().(DecryptMessage)
		return &v
	case 52:
		v := reflect.Indirect(reflect.New(t)).Interface().(DecryptedMessage)
		return &v
	case 53:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignIdentity)
		return &v
	case 54:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignedIdentity)
		return &v
	case 55:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetFeatures)
		return &v
	case 56:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaGetAddress)
		return &v
	case 57:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaAddress)
		return &v
	case 58:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaSignTx)
		return &v
	case 59:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaTxRequest)
		return &v
	case 60:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaTxAck)
		return &v
	case 61:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetECDHSessionKey)
		return &v
	case 62:
		v := reflect.Indirect(reflect.New(t)).Interface().(ECDHSessionKey)
		return &v
	case 63:
		v := reflect.Indirect(reflect.New(t)).Interface().(SetU2FCounter)
		return &v
	case 64:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaSignMessage)
		return &v
	case 65:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaVerifyMessage)
		return &v
	case 66:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaMessageSignature)
		return &v
	case 100:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkDecision)
		return &v
	case 101:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkGetState)
		return &v
	case 102:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkState)
		return &v
	case 103:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkStop)
		return &v
	case 104:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkLog)
		return &v
	case 110:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemoryRead)
		return &v
	case 111:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemory)
		return &v
	case 112:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemoryWrite)
		return &v
	case 113:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkFlashErase)
		return &v
	}
	return nil
}

func getWireType(ftype string) (res uint) {
	if strings.HasPrefix(ftype, "uint") {
		res = 0
	} else if strings.HasPrefix(ftype, "int") {
		res = 0
	} else if ftype == "bool" {
		res = 0
	} else if ftype == "[]uint8" { // bytes
		res = 2
	} else if ftype == "string" { // UnicodeType
		res = 2
	} else if ftype == "struct {}" {
		res = 2
	}
	return
}

func LoadMessage(reader io.Reader, msgType MessageType) (interface{}, error) {
	// tname := MessageType_name[msgType]
	// tname = strings.Split(tname, "_")[1]
	// typeName := "trezor." + tname
	//t := proto.MessageType(typeName).Elem()
	//tt := proto.MessageType(typeName)
	//v := reflect.Indirect(reflect.New(t)).Interface()

	fields := make(map[uint]ProtoField)
	target := getEmptyObj(msgType)
	v := reflect.ValueOf(target).Elem()

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Reflection Error")
	}

	for i := 0; i < v.NumField(); i++ {
		f := ProtoField{}
		f.field = v.Field(i)
		tags := v.Type().Field(i).Tag.Get("protobuf")
		if len(tags) == 0 {
			continue
		}
		tagArray := strings.Split(tags, ",")
		f.ftype = f.field.Type().String()
		key, _ := strconv.Atoi(tagArray[1])
		for _, tag := range tagArray {
			if tag == "rep" {
				f.isArray = true
				break
			}
		}
		fields[uint(key)] = f
	}

	fmt.Printf(">>>>>>>>>>>FIELDS>>>>>>>>>>>> %v, %v\n", msgType, fields)
	for {
		fkey, err := loadUvarint(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		ftag := fkey >> 3
		wtype := fkey & 7

		var field ProtoField
		var ok bool
		fmt.Printf(">>>>>>>>>>>HERE>>>>>>>>>>>> %v, %v, %v\n", ftag, wtype, fields[ftag])
		if field, ok = fields[ftag]; !ok {
			if wtype == 0 {
				loadUvarint(reader)
			} else if wtype == 2 {
				ivalue, err := loadUvarint(reader)
				if err != nil {
					return nil, err
				}

				buffer := make([]byte, ivalue)
				_, err = reader.Read(buffer)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Unknow protobuf field")
			}
			continue
		}

		//     fname, ftype, fflags = field
		if wtype != getWireType(field.ftype) {
			return nil, fmt.Errorf("Parsed wire type differs from the schema")
		}

		ivalue, err := loadUvarint(reader)
		if err != nil {
			// fmt.Printf(">>>>>>>>>>>IVALUE ERROR>>>>>>>>>>>> %v\n", err)
			return nil, err
		}
		fmt.Printf(">>>>>>>>>>>IVALUE>>>>>>>>>>>>ivalue: %v, ftype: %v, isArray: %v\n", ivalue, field.ftype, field.isArray)

		if field.ftype == "int8" {

		} else if field.ftype == "int16" {

		} else if field.ftype == "int32" {

		} else if field.ftype == "int64" {

		} else if field.ftype == "bool" {

		} else if field.ftype == "[]uint8" { // bytes

		} else if field.ftype == "string" { // UnicodeType

		} else if field.ftype == "struct {}" {
			//TODO
		} else {

		}

		if strings.HasPrefix(field.ftype, "uint") {
			if field.isArray {
				i := field.field.Interface()
				if field.ftype == "[]uint8" {
					u := i.([]uint8)
					u = append(u, uint8(ivalue))
					field.field.Set(reflect.ValueOf(u))
				} else if field.ftype == "uint16" {
					u := i.([]uint16)
					u = append(u, uint16(ivalue))
					field.field.Set(reflect.ValueOf(u))
				} else if field.ftype == "uint32" {
					u := i.([]uint32)
					u = append(u, uint32(ivalue))
					field.field.Set(reflect.ValueOf(u))
				} else if field.ftype == "uint64" {
					u := i.([]uint64)
					u = append(u, uint64(ivalue))
					field.field.Set(reflect.ValueOf(u))
				}
			} else {
				field.field.SetUint(uint64(ivalue))
			}
		} else if strings.HasPrefix(field.ftype, "int") {
			if field.isArray {
				i := field.field.Interface()
				if field.ftype == "[]int8" {
					n := i.([]int8)
					n = append(n, int8(ivalue))
					field.field.Set(reflect.ValueOf(n))
				} else if field.ftype == "int16" {
					n := i.([]int16)
					n = append(n, int16(ivalue))
					field.field.Set(reflect.ValueOf(n))
				} else if field.ftype == "int32" {
					n := i.([]int32)
					n = append(n, int32(ivalue))
					field.field.Set(reflect.ValueOf(n))
				} else if field.ftype == "int64" {
					n := i.([]int64)
					n = append(n, int64(ivalue))
					field.field.Set(reflect.ValueOf(n))
				}
			} else {
				// field.field.SetInt(int64(uint2Sint(ivalue)))
				field.field.SetInt(int64(ivalue)) // TODO
			}
		} else if field.ftype == "bool" {
			if field.isArray {
				i := field.field.Interface()
				b := i.([]bool)
				b = append(b, ivalue != 0)
				field.field.Set(reflect.ValueOf(b))
			} else {
				field.field.SetBool(ivalue != 0)
			}
		} else if field.ftype == "[]uint8" { // bytes
			bytes := make([]byte, ivalue)
			_, err = reader.Read(bytes)
			if err != nil {
				return nil, err
			}

			if field.isArray {
				i := field.field.Interface()
				s := i.([][]uint8)
				s = append(s, bytes)
				field.field.Set(reflect.ValueOf(s))
			} else {
				field.field.SetBytes(bytes)
			}

			fmt.Printf(">>>>>>>>>>> FIELD (bytes) >>>>>>>>>>>> %v\n", field.field)
		} else if field.ftype == "string" { // UnicodeType
			bytes := make([]byte, ivalue)
			_, err = reader.Read(bytes)
			if err != nil {
				return nil, err
			}

			if field.isArray {
				i := field.field.Interface()
				s := i.([]string)
				s = append(s, string(bytes))
				field.field.Set(reflect.ValueOf(s))
			} else {
				field.field.SetString(string(bytes))
			}
		} else if field.ftype == "struct {}" {
			/*********** TEMP **********/
			if wtype == 0 {
				loadUvarint(reader)
			} else if wtype == 2 {
				ivalue, err := loadUvarint(reader)
				if err != nil {
					return nil, err
				}

				buffer := make([]byte, ivalue)
				_, err = reader.Read(buffer)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Unknow protobuf field")
			}
			/*********** TEMP **********/

			// field.field.Set()
			// field.fvalue = load_message(LimitedReader(reader, ivalue), ftype)
		} else {
			fmt.Printf("=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-= unknown ftype: %v\n", field.ftype)
			field.field.SetInt(int64(ivalue))
		}
	}
	fmt.Printf(">>>>>>>>>>>LOADED MESSAGE>>>>>>>>>>>> %v\n", target)
	return target, nil
}

func DumpMessage(writer io.Writer, msg interface{}) error {
	v := reflect.ValueOf(msg).Elem()
	fmt.Printf("<<<<<<<<<<<<< DUMP MESSAGE >>>>>>>>>>>> %v\n", v.NumField())
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if field.Interface() == nil {
			continue
		}

		tags := v.Type().Field(i).Tag.Get("protobuf")
		if len(tags) == 0 {
			continue
		}
		tagArray := strings.Split(tags, ",")

		nameStartIdx := strings.Index(tags, "name=") + len("name=")
		fname := tags[nameStartIdx:]
		nameEndIdx := strings.Index(fname, ",")
		fname = fname[:nameEndIdx]

		ftype := field.Type().String() // tagArray[0]
		var isArray bool
		for _, tag := range tagArray {
			if tag == "rep" {
				isArray = true
				break
			}
		}
		ftag, _ := strconv.Atoi(tagArray[1])
		fkey := (uint(ftag) << 3) | getWireType(ftype)

		fmt.Printf("<<<<<<<<<<<<< >>>>>>>>>>>> ftype: %v, isArray: %v\n", ftype, isArray)

		var repvalue []interface{}
		var stype string
		if !isArray {
			stype = ftype
			repvalue = append(repvalue, field.Interface())
		} else {
			if ftype[:2] != "[]" {
				return fmt.Errorf("It's not an array type with the 'rep' tag!")
			}
			stype = ftype[2:]

			if stype == "uint8" {
				// repvalue = append(repvalue, field.Interface().([]uint8)[:])
				arr := field.Interface().([]uint8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint16" {
				// repvalue = append(repvalue, field.Interface().([]uint16))
				arr := field.Interface().([]uint16)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint32" {
				// repvalue = append(repvalue, field.Interface().([]uint32))
				arr := field.Interface().([]uint32)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint64" {
				// repvalue = append(repvalue, field.Interface().([]uint64))
				arr := field.Interface().([]uint64)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int8" {
				// repvalue = append(repvalue, field.Interface().([]int8))
				arr := field.Interface().([]int8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int16" {
				// repvalue = append(repvalue, field.Interface().([]int16))
				arr := field.Interface().([]int16)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int32" {
				// repvalue = append(repvalue, field.Interface().([]int32))
				arr := field.Interface().([]uint32)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int64" {
				// repvalue = append(repvalue, field.Interface().([]int64))
				arr := field.Interface().([]int64)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "bool" {
				// repvalue = append(repvalue, field.Interface().([]bool))
				arr := field.Interface().([]bool)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "[]uint8" { // bytes
				// repvalue = append(repvalue, field.Interface().([][]uint8))
				arr := field.Interface().([][]uint8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "string" { // UnicodeType
				// repvalue = append(repvalue, field.Interface().([]string))
				arr := field.Interface().([]string)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "struct {}" {
				//TODO
			} else {
				//TODO
				fmt.Printf("############### stype: %v\n", stype)
				continue
			}
		}

		// fmt.Printf("<<<<<<<<<<<<< fname: %v, ftag: %v, WType: %v, fkey: %v, ftype: %v, isArray: %v\n", fname, ftag, getWireType(ftype), fkey, ftype, isArray)
		// fmt.Printf("<<<<<<<<<<<<< repvalue size >>>>>>>>>>>> %v\n", len(repvalue))
		for _, svalue := range repvalue {
			dumpUvarint(writer, fkey)

			if stype == "uint8" {
				dumpUvarint(writer, uint(svalue.(uint8)))
			} else if stype == "uint16" {
				dumpUvarint(writer, uint(svalue.(uint16)))
			} else if stype == "uint32" {
				dumpUvarint(writer, uint(svalue.(uint32)))
			} else if stype == "uint64" {
				dumpUvarint(writer, uint(svalue.(uint64)))
			} else if stype == "int8" {
				dumpUvarint(writer, sint2Uint(int(svalue.(int8))))
			} else if stype == "int16" {
				dumpUvarint(writer, sint2Uint(int(svalue.(int16))))
			} else if stype == "int32" {
				dumpUvarint(writer, sint2Uint(int(svalue.(int32))))
			} else if stype == "int64" {
				dumpUvarint(writer, sint2Uint(int(svalue.(int64))))
			} else if stype == "bool" {
				var b uint
				if svalue.(bool) {
					b = 1
				}
				dumpUvarint(writer, b)
			} else if stype == "[]uint8" { // bytes
				dumpUvarint(writer, uint(len(svalue.([]byte))))
				writer.Write(svalue.([]byte))
			} else if stype == "string" { // UnicodeType
				dumpUvarint(writer, uint(len(svalue.(string))))
				writer.Write([]byte(svalue.(string)))
			} else if stype == "struct {}" {
				//TODO
			} else {
				//TODO
				fmt.Printf("############### stype: %v\n", stype)
				continue
			}
		}
	}
	return nil
}
