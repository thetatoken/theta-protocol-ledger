package trezor

import (
	"errors"
	fmt "fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

var ErrKeyNotFound = errors.New("KeyNotFound")

func loadUvarint(reader io.Reader, limit *int32) (uint, error) {
	buffer := make([]byte, 1)
	result := uint(0)
	shift := uint(0)
	byt := byte(0x80)
	for {
		res := byt & 0x80
		if res == 0 {
			break
		}
		if *limit <= 0 {
			return 0, fmt.Errorf("read limit exceeded")
		}
		_, err := reader.Read(buffer)
		if err != nil {
			return 0, err
		}
		*limit--
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
	pbtype  string
	fname   string
	isArray bool
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

// func LoadMessage(reader io.Reader, msgType MessageType, limit *int32) (interface{}, error) {
func LoadMessage(reader io.Reader, target interface{}, limit *int32) (interface{}, error) {
	// tname := MessageType_name[msgType]
	// tname = strings.Split(tname, "_")[1]
	// typeName := "trezor." + tname
	//t := proto.MessageType(typeName).Elem()
	//tt := proto.MessageType(typeName)
	//v := reflect.Indirect(reflect.New(t)).Interface()

	fields := make(map[uint]ProtoField)
	v := reflect.ValueOf(target).Elem()

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Reflection Error")
	}

	for i := 0; i < v.NumField(); i++ {
		f := ProtoField{}
		f.field = v.Field(i)
		f.ftype = f.field.Type().String()
		tags := v.Type().Field(i).Tag.Get("protobuf")
		if len(tags) == 0 {
			continue
		}
		tagArray := strings.Split(tags, ",")
		f.pbtype = tagArray[0]
		key, _ := strconv.Atoi(tagArray[1])
		fields[uint(key)] = f
		for _, tag := range tagArray {
			if tag == "rep" {
				f.isArray = true
				break
			}
		}
	}

	for {
		if *limit <= 0 {
			return nil, fmt.Errorf("read limit exceeded")
		}
		fkey, err := loadUvarint(reader, limit)
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
		if field, ok = fields[ftag]; !ok {
			if wtype == 0 {
				loadUvarint(reader, limit)
			} else if wtype == 2 {
				ivalue, err := loadUvarint(reader, limit)
				if err != nil {
					return nil, err
				}

				buffer := make([]byte, ivalue)
				_, err = reader.Read(buffer)
				if err != nil {
					return nil, err
				}
				*limit -= int32(ivalue)
			} else {
				return nil, fmt.Errorf("Unknow protobuf field")
			}
			continue
		}

		if wtype != getWireType(field.ftype) {
			return nil, fmt.Errorf("Parsed wire type differs from the schema")
		}

		ivalue, err := loadUvarint(reader, limit)
		if err != nil {
			return nil, err
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
				field.field.SetInt(int64(uint2Sint(ivalue)))
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
			*limit -= int32(ivalue)

			if field.isArray {
				i := field.field.Interface()
				s := i.([][]uint8)
				s = append(s, bytes)
				field.field.Set(reflect.ValueOf(s))
			} else {
				field.field.SetBytes(bytes)
			}
		} else if field.ftype == "string" { // UnicodeType
			bytes := make([]byte, ivalue)
			_, err = reader.Read(bytes)
			if err != nil {
				return nil, err
			}
			*limit -= int32(ivalue)

			if field.isArray {
				i := field.field.Interface()
				s := i.([]string)
				s = append(s, string(bytes))
				field.field.Set(reflect.ValueOf(s))
			} else {
				field.field.SetString(string(bytes))
			}
		} else if field.ftype == "struct {}" {
			// skip it for now
			if wtype == 0 {
				loadUvarint(reader, limit)
			} else if wtype == 2 {
				ivalue, err := loadUvarint(reader, limit)
				if err != nil {
					return nil, err
				}

				buffer := make([]byte, ivalue)
				_, err = reader.Read(buffer)
				if err != nil {
					return nil, err
				}
				*limit -= int32(ivalue)
			} else {
				return nil, fmt.Errorf("Unknow field wtype")
			}
		} else {
			if field.pbtype == "varint" {
				field.field.SetInt(int64(ivalue))
			} else {
				// skip it for now
				if wtype == 0 {
					loadUvarint(reader, limit)
				} else if wtype == 2 {
					ivalue, err := loadUvarint(reader, limit)
					if err != nil {
						return nil, err
					}

					buffer := make([]byte, ivalue)
					_, err = reader.Read(buffer)
					if err != nil {
						return nil, err
					}
					*limit -= int32(ivalue)
				} else {
					return nil, fmt.Errorf("Unknow field wtype")
				}
			}
		}
	}

	return target, nil
}

func DumpMessage(writer io.Writer, msg interface{}) error {
	v := reflect.ValueOf(msg).Elem()
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
				arr := field.Interface().([]uint8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint16" {
				arr := field.Interface().([]uint16)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint32" {
				arr := field.Interface().([]uint32)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "uint64" {
				arr := field.Interface().([]uint64)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int8" {
				arr := field.Interface().([]int8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int16" {
				arr := field.Interface().([]int16)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int32" {
				arr := field.Interface().([]uint32)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "int64" {
				arr := field.Interface().([]int64)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "bool" {
				arr := field.Interface().([]bool)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "[]uint8" { // bytes
				arr := field.Interface().([][]uint8)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "string" { // UnicodeType
				arr := field.Interface().([]string)[:]
				for _, n := range arr {
					repvalue = append(repvalue, n)
				}
			} else if stype == "struct {}" {
				//TODO
			} else {
				//TODO
				continue
			}
		}

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
				continue
			}
		}
	}
	return nil
}
