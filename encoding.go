package milter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

func decode(data []byte, dest interface{}) error {
	d := &decoder{data}
	return d.decode(dest)
}

// A decoder decodes data from the milter protocol's binary wire format.
type decoder struct {
	data []byte
}

var errNotEnoughData = errors.New("not enough data")

func (d *decoder) decode(val interface{}) error {
	var v reflect.Value
	switch val := val.(type) {
	case reflect.Value:
		v = val
	default:
		v = reflect.ValueOf(val)
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return d.decode(v.Elem())

	case reflect.Uint8:
		if len(d.data) < 1 {
			return errNotEnoughData
		}
		v.SetUint(uint64(d.data[0]))
		d.data = d.data[1:]
		return nil

	case reflect.Uint16:
		if len(d.data) < 2 {
			return errNotEnoughData
		}
		v.SetUint(uint64(binary.BigEndian.Uint16(d.data)))
		d.data = d.data[2:]
		return nil

	case reflect.Uint32:
		if len(d.data) < 4 {
			return errNotEnoughData
		}
		v.SetUint(uint64(binary.BigEndian.Uint32(d.data)))
		d.data = d.data[4:]
		return nil

	case reflect.String:
		i := bytes.IndexByte(d.data, 0)
		if i == -1 {
			return errors.New("unterminated C string")
		}
		v.SetString(string(d.data[:i]))
		d.data = d.data[i+1:]
		return nil

	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			if err := d.decode(v.Field(i)); err != nil {
				return err
			}
		}
		return nil

	default:
		panic(fmt.Errorf("decode: unsupported type: %T", v.Interface()))
	}
}

func encode(val interface{}) []byte {
	e := &encoder{new(bytes.Buffer)}
	e.encode(val)
	return e.Bytes()
}

// An encoder encodes data into the milter protocol's binary wire format.
type encoder struct {
	*bytes.Buffer
}

func (e *encoder) encode(val interface{}) {
	var v reflect.Value
	switch val := val.(type) {
	case reflect.Value:
		v = val
	default:
		v = reflect.ValueOf(val)
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		e.encode(v.Elem())

	case reflect.Uint8:
		e.WriteByte(byte(v.Uint()))

	case reflect.Uint16:
		binary.Write(e, binary.BigEndian, uint16(v.Uint()))

	case reflect.Uint32:
		binary.Write(e, binary.BigEndian, uint32(v.Uint()))

	case reflect.String:
		e.WriteString(v.String())
		e.WriteByte(0)

	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			e.encode(v.Field(i))
		}

	default:
		panic(fmt.Errorf("encode: unsupported type: %T", v.Interface()))
	}
}
