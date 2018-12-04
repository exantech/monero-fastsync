package moneroproto

import (
	"bytes"
	"errors"
	"io"
	"log"
	"reflect"
)

//TODO: rename it to EncodeMessage
func Write(writer io.Writer, obj interface{}) error {
	_, err := writer.Write(MessagePreamble)
	if err != nil {
		return err
	}

	return Encode(writer, obj)
}

func Encode(writer io.Writer, obj interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return doEncode(writer, v, 0)
}

func doEncode(writer io.Writer, value reflect.Value, level int) error {
	switch value.Kind() {
	case reflect.Invalid:
		return errors.New("invalid type")
	case reflect.Bool:
		_, err := writeBool(writer, value.Bool())
		return err
	case reflect.Int64:
		_, err := writeInt64(writer, value.Int())
		return err
	case reflect.Int:
		_, err := writeInt32(writer, int32(value.Int()))
		return err
	case reflect.Int32:
		_, err := writeInt32(writer, int32(value.Int()))
		return err
	case reflect.Int16:
		_, err := writeInt16(writer, int16(value.Int()))
		return err
	case reflect.Int8:
		_, err := writeInt8(writer, int8(value.Int()))
		return err
	case reflect.Uint64:
		_, err := writeUint64(writer, value.Uint())
		return err
	case reflect.Uint:
		_, err := writeUint32(writer, uint32(value.Uint()))
		return err
	case reflect.Uint32:
		_, err := writeUint32(writer, uint32(value.Uint()))
		return err
	case reflect.Uint16:
		_, err := writeUint16(writer, uint16(value.Uint()))
		return err
	case reflect.Uint8:
		_, err := writeUint8(writer, uint8(value.Uint()))
		return err
	case reflect.Float64:
		_, err := writeFloat64(writer, value.Float())
		return err
	case reflect.Ptr:
		return doEncode(writer, value.Elem(), level)
	case reflect.Slice:
		return encodeArray(writer, value)
	case reflect.Struct:
		return encodeObject(writer, value, level)
	default:
		//currently unsupported types: Array, Chan, Func, Interface, Map, String, UnsafePointer
		log.Fatal("unsupported type: ", value.Kind())
	}

	return nil
}

func encodeObject(writer io.Writer, value reflect.Value, level int) error {
	if level != 0 {
		_, err := writeObjectTag(writer)
		if err != nil {
			return err
		}
	}

	fields := value.NumField()
	_, err := packVarint(writer, uint64(fields))
	if err != nil {
		return err
	}

	for i := 0; i < fields; i++ {
		name := value.Type().Field(i).Tag.Get("monerobinkv")
		if len(name) == 0 {
			continue
		}

		_, err := writeName(writer, []byte(name))
		if err != nil {
			return err
		}

		err = doEncode(writer, value.Field(i), level+1)
		if err != nil {
			return err
		}
	}

	return nil
}

func encodeArray(writer io.Writer, value reflect.Value) error {
	if value.Kind() != reflect.Slice {
		//programming error
		//TODO: remove log.fatals with panics
		panic("value must be a slice")
	}

	//TODO: replace errors.New with error objects
	elemType := getWireObjectType(value)
	if elemType == TypeUint8 {
		// encode []byte as binary string
		elemType = TypeBinaryString
	} else {
		elemType |= FlagArray
	}

	_, err := writeType(writer, elemType)
	if err != nil {
		return err
	}

	_, err = packVarint(writer, uint64(value.Len()))
	if err != nil {
		return err
	}

	for i := 0; i < value.Len(); i++ {
		elem := value.Index(i)
		err = encodeArrayElement(writer, elem)
		if err != nil {
			return err
		}
	}

	return nil
}

func encodeArrayElement(writer io.Writer, value reflect.Value) error {
	//if value.Kind() == reflect.Slice {
	//	panic("writing slice of slices is currently not supported")
	//}

	if value.Kind() == reflect.Ptr {
		return encodeArrayElement(writer, value.Elem())
	}

	if value.Kind() == reflect.Struct {
		return encodeObject(writer, value, 0)
	}

	switch value.Kind() {
	case reflect.Bool:
		_, err := writeBoolBlob(writer, value.Bool())
		return err
	case reflect.Int64:
		_, err := writeInt64Blob(writer, value.Int())
		return err
	case reflect.Int:
		_, err := writeInt32Blob(writer, int32(value.Int()))
		return err
	case reflect.Int32:
		_, err := writeInt32Blob(writer, int32(value.Int()))
		return err
	case reflect.Int16:
		_, err := writeInt16Blob(writer, int16(value.Int()))
		return err
	case reflect.Int8:
		_, err := writeInt8Blob(writer, int8(value.Int()))
		return err
	case reflect.Uint64:
		_, err := writeUint64Blob(writer, value.Uint())
		return err
	case reflect.Uint:
		_, err := writeUint32Blob(writer, uint32(value.Uint()))
		return err
	case reflect.Uint32:
		_, err := writeUint32Blob(writer, uint32(value.Uint()))
		return err
	case reflect.Uint16:
		_, err := writeUint16Blob(writer, uint16(value.Uint()))
		return err
	case reflect.Uint8:
		_, err := writeUint8Blob(writer, uint8(value.Uint()))
		return err
	case reflect.Float64:
		_, err := writeFloat64Blob(writer, value.Float())
		return err
	case reflect.Slice:
		_, err := packVarint(writer, uint64(value.Len()))
		if err != nil {
			return err
		}
		_, err = writeBlob(writer, value.Bytes())
		return err
	default:
		panic("unsupported array element type")
	}
}

func getWireObjectType(value reflect.Value) byte {
	var elem reflect.Value
	if value.Kind() == reflect.Slice {
		if value.Len() > 0 {
			elem = value.Index(0)
		} else {
			slice := reflect.MakeSlice(value.Type(), 1, 1)
			elem = slice.Index(0)
		}
	} else if value.Kind() == reflect.Ptr {
		elem = value.Elem()
	} else {
		//programming error
		panic("value must be a slice or a pointer")
	}

	switch elem.Kind() {
	case reflect.Bool:
		return TypeBool
	case reflect.Int64:
		return TypeInt64
	case reflect.Int:
		return TypeInt32
	case reflect.Int32:
		return TypeInt32
	case reflect.Int16:
		return TypeInt16
	case reflect.Int8:
		return TypeInt8
	case reflect.Uint64:
		return TypeUint64
	case reflect.Uint:
		return TypeUint32
	case reflect.Uint32:
		return TypeUint32
	case reflect.Uint16:
		return TypeUint16
	case reflect.Uint8:
		return TypeUint8
	case reflect.Float64:
		return TypeDouble
	case reflect.Ptr:
		return getWireObjectType(elem)
	case reflect.Slice:
		//TODO: make recursive check if elem is of type []byte
		return TypeBinaryString
	case reflect.Struct:
		return TypeObject
	default:
		log.Fatal("unsupported type: ", value.Kind())
	}

	//shouldn't be reached
	return 0
}

//TODO: rename it to DecodeMessage
func Read(reader io.Reader, obj interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return errors.New("object is expected to be a pointer")
	}

	if v.IsNil() {
		return errors.New("nil pointer passed")
	}

	preamble := make([]byte, len(MessagePreamble))
	_, err := io.ReadFull(reader, preamble)
	if err == io.EOF {
		return ErrUnexpectedEof
	}

	if err != nil {
		return err
	}

	if !bytes.Equal(preamble, MessagePreamble) {
		return errors.New("message preamble mismatch")
	}

	return decodeObject(reader, v.Elem())
}

func decodeObject(reader io.Reader, v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		return errors.New("value is not a struct")
	}

	fields := structFields(v)
	size, err := unpackVarint(reader)
	if err == io.EOF {
		return ErrUnexpectedEof
	}

	if err != nil {
		return err
	}

	for i := uint64(0); i < size; i++ {
		name, err := readName(reader)
		if err == io.EOF {
			return ErrUnexpectedEof
		}

		if err != nil {
			return err
		}

		f, ok := fields[string(name)]
		if !ok {
			return errors.New("unexpected field name")
		}

		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}

		err = doDecode(reader, f)
		if err == io.EOF && i < size - 1 {
			return ErrUnexpectedEof
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func doDecode(reader io.Reader, v reflect.Value) error {
	t, err := readType(reader)
	if err == io.EOF {
		return ErrUnexpectedEof
	}

	if err != nil {
		return err
	}

	//TODO: check type
	if t&FlagArray != 0 {
		if v.Kind() != reflect.Slice {
			return errors.New("unexpected array occured")
		}

		return decodeArray(reader, t, v)
	}

	return decodeValue(reader, t, v)
}

func decodeValue(reader io.Reader, valueType byte, v reflect.Value) error {
	var err error
	switch valueType {
	case TypeInt64:
		if v.Kind() != reflect.Int64 {
			return errors.New("type mismatch")
		}
		var val int64
		val, err = readInt64(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetInt(val)
	case TypeInt32:
		if v.Kind() != reflect.Int32 && v.Kind() != reflect.Int {
			return errors.New("type mismatch")
		}
		var val int32
		val, err = readInt32(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetInt(int64(val))
	case TypeInt16:
		if v.Kind() != reflect.Int16 {
			return errors.New("type mismatch")
		}
		var val int16
		val, err = readInt16(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetInt(int64(val))
	case TypeInt8:
		if v.Kind() != reflect.Int8 {
			return errors.New("type mismatch")
		}
		var val int8
		val, err = readInt8(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetInt(int64(val))
	case TypeUint64:
		if v.Kind() != reflect.Uint64 {
			return errors.New("type mismatch")
		}
		var val uint64
		val, err = readUint64(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetUint(val)
	case TypeUint32:
		if v.Kind() != reflect.Uint32 && v.Kind() != reflect.Uint {
			return errors.New("type mismatch")
		}
		var val uint32
		val, err = readUint32(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetUint(uint64(val))
	case TypeUint16:
		if v.Kind() != reflect.Uint16 {
			return errors.New("type mismatch")
		}
		var val uint16
		val, err = readUint16(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetUint(uint64(val))
	case TypeUint8:
		if v.Kind() != reflect.Uint8 {
			return errors.New("type mismatch")
		}
		var val uint8
		val, err = readUint8(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetUint(uint64(val))
	case TypeDouble:
		if v.Kind() != reflect.Float64 {
			return errors.New("type mismatch")
		}
		var val float64
		val, err = readFloat64(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetFloat(val)
	case TypeBinaryString:
		if v.Kind() != reflect.Slice {
			return errors.New("type mismatch")
		}
		var data []byte
		data, err = readBinaryString(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.Set(reflect.ValueOf(data))
	case TypeBool:
		if v.Kind() != reflect.Bool {
			return errors.New("type mismatch")
		}
		var val bool
		val, err := readBool(reader)
		if err != nil && err != io.EOF {
			return err
		}
		v.SetBool(val)
	case TypeObject:
		if v.Kind() != reflect.Struct {
			return errors.New("type mismatch")
		}
		err = decodeObject(reader, v)
		if err != nil && err != io.EOF {
			return err
		}
	default:
		log.Fatal("not implemented yet")
	}

	return err
}

func structFields(v reflect.Value) map[string]reflect.Value {
	if v.Kind() != reflect.Struct {
		log.Fatal("v is expected to be a struct")
	}

	fields := make(map[string]reflect.Value)
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Tag.Get("monerobinkv")
		if len(name) == 0 {
			continue
		}

		fields[name] = v.Field(i)
	}

	return fields
}

func decodeArray(reader io.Reader, arrayType byte, value reflect.Value) error {
	size, err := unpackVarint(reader)
	if err == io.EOF {
		return ErrUnexpectedEof
	}

	if err != nil {
		return err
	}

	newv := makeSlice(value, int(size))
	value.Set(newv)

	elemType := arrayType & ^FlagArray

	for i := 0; i < int(size); i++ {
		elem := value.Index(i)
		err = decodeValue(reader, elemType, elem)
		if err == io.EOF && i < int(size) - 1 {
			return ErrUnexpectedEof
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func makeSlice(value reflect.Value, size int) reflect.Value {
	if value.Kind() != reflect.Slice {
		log.Fatal("value expected to be a slice")
	}

	capacity := size + size/2
	if capacity < 4 {
		capacity = 4
	}

	return reflect.MakeSlice(value.Type(), size, capacity)
}
