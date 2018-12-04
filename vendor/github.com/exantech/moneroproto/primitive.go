package moneroproto

import (
	"errors"
	"io"
	"unsafe"
)

var (
	ErrVarintTooBig   = errors.New("varint value too big")
	ErrNotEnoughData  = errors.New("not enough data in read buffer")
	ErrUnexpectedType = errors.New("unexpected type")
	ErrUnexpectedEof  = errors.New("stream unexpectedly ended")
)

func checkEof(err error, read, required int) error {
	if err != io.EOF {
		return err
	}

	if required < read {
		return ErrUnexpectedEof
	}

	return err
}

func uint64ToBytes(val uint64) []byte {
	bytes := make([]byte, 8)
	*(*uint64)(unsafe.Pointer(&bytes[0])) = val
	return bytes
}

//buf must be 8 bytes long
func bytesToUint64(buf []byte) uint64 {
	return *(*uint64)(unsafe.Pointer(&buf[0]))
}

func uint32ToBytes(val uint32) []byte {
	bytes := make([]byte, 4)
	*(*uint32)(unsafe.Pointer(&bytes[0])) = val
	return bytes
}

//buf must be 4 bytes long
func bytesToUint32(buf []byte) uint32 {
	return *(*uint32)(unsafe.Pointer(&buf[0]))
}

//buf must be 2 bytes long
func bytesToUint16(buf []byte) uint16 {
	return *(*uint16)(unsafe.Pointer(&buf[0]))
}

func int64ToBytes(val int64) []byte {
	bytes := make([]byte, 8)
	*(*int64)(unsafe.Pointer(&bytes[0])) = val
	return bytes
}

//buf must be 8 bytes long
func bytesToInt64(buf []byte) int64 {
	return *(*int64)(unsafe.Pointer(&buf[0]))
}

func int32ToBytes(val int32) []byte {
	bytes := make([]byte, 4)
	*(*int32)(unsafe.Pointer(&bytes[0])) = val
	return bytes
}

//buf must be 4 bytes long
func bytesToInt32(buf []byte) int32 {
	return *(*int32)(unsafe.Pointer(&buf[0]))
}

//buf must be 2 bytes long
func bytesToInt16(buf []byte) int16 {
	return *(*int16)(unsafe.Pointer(&buf[0]))
}

func float64ToBytes(val float64) []byte {
	bytes := make([]byte, unsafe.Sizeof(val))
	*(*float64)(unsafe.Pointer(&bytes[0])) = val
	return bytes
}

//buf must be 8 bytes long
func bytesToFloat64(buf []byte) float64 {
	return *(*float64)(unsafe.Pointer(&buf[0]))
}

func packVarint(writer io.Writer, val uint64) (int, error) {
	r := val << 2
	bytes := 0
	switch {
	case val <= 63:
		r |= uint64(MarkByte)
		bytes = 1
	case val <= 16383:
		r |= uint64(MarkWord)
		bytes = 2
	case val <= 1073741823:
		r |= uint64(MarkDWord)
		bytes = 4
	case val <= 4611686018427387903:
		r |= uint64(MarkInt64)
		bytes = 8
	default:
		return 0, ErrVarintTooBig
	}

	return writer.Write(uint64ToBytes(r)[0:bytes])
}

func unpackVarint(reader io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	r, err := reader.Read(buf[0:1])

	if err != nil && err != io.EOF {
		return 0, err
	}

	if r == 0 {
		return 0, ErrNotEnoughData
	}

	need := 0
	mask := buf[0] & MarkMask
	switch mask {
	case MarkByte:
	case MarkWord:
		need = 2
	case MarkDWord:
		need = 4
	case MarkInt64:
		need = 8
	}

	if need == 0 {
		return bytesToUint64(buf) >> 2, nil
	}

	r, err = io.ReadFull(reader, buf[1:need])
	err = checkEof(err, r, len(buf[1:need]))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToUint64(buf) >> 2, err
}

func writeType(writer io.Writer, t byte) (int, error) {
	return writer.Write([]byte{t})
}

func readType(reader io.Reader) (byte, error) {
	t := []byte{0}
	_, err := reader.Read(t)
	return t[0], err
}

func writeUint64(writer io.Writer, val uint64) (int, error) {
	typ := []byte{TypeUint64}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeUint64Blob(writer, val)
	return wrote + 1, err
}

func writeUint64Blob(writer io.Writer, val uint64) (int, error) {
	return writer.Write(uint64ToBytes(val))
}

func readUint64(reader io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToUint64(buf), err
}

func writeUint32(writer io.Writer, val uint32) (int, error) {
	typ := []byte{TypeUint32}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeUint32Blob(writer, val)
	return wrote + 1, err
}

func writeUint32Blob(writer io.Writer, val uint32) (int, error) {
	return writer.Write(uint32ToBytes(val))
}

func readUint32(reader io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToUint32(buf), err
}

func writeUint16(writer io.Writer, val uint16) (int, error) {
	typ := []byte{TypeUint16}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeUint16Blob(writer, val)
	return wrote + 1, err
}

func writeUint16Blob(writer io.Writer, val uint16) (int, error) {
	buf := []byte{byte(val & 0xFF), byte(val>>8) & 0xFF}
	return writer.Write(buf)
}

func readUint16(reader io.Reader) (uint16, error) {
	buf := make([]byte, 2)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToUint16(buf), err
}

func writeUint8(writer io.Writer, val uint8) (int, error) {
	typ := []byte{TypeUint8}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeUint8Blob(writer, val)
	return wrote + 1, err
}

func writeUint8Blob(writer io.Writer, val uint8) (int, error) {
	buf := []byte{byte(val)}
	return writer.Write(buf)
}

func readUint8(reader io.Reader) (uint8, error) {
	buf := make([]byte, 1)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return uint8(buf[0]), err
}

func writeInt64(writer io.Writer, val int64) (int, error) {
	typ := []byte{TypeInt64}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeInt64Blob(writer, val)
	return wrote + 1, err
}

func writeInt64Blob(writer io.Writer, val int64) (int, error) {
	return writer.Write(int64ToBytes(val))
}

func readInt64(reader io.Reader) (int64, error) {
	buf := make([]byte, 8)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToInt64(buf), err
}

func writeInt32(writer io.Writer, val int32) (int, error) {
	typ := []byte{TypeInt32}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeInt32Blob(writer, val)
	return wrote + 1, err
}

func writeInt32Blob(writer io.Writer, val int32) (int, error) {
	return writer.Write(int32ToBytes(val))
}

func readInt32(reader io.Reader) (int32, error) {
	buf := make([]byte, 4)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToInt32(buf), err
}

func writeInt16(writer io.Writer, val int16) (int, error) {
	typ := []byte{TypeInt16}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeInt16Blob(writer, val)
	return wrote + 1, err
}

func writeInt16Blob(writer io.Writer, val int16) (int, error) {
	buf := []byte{byte(val & 0xFF), byte(val>>8) & 0xFF}
	return writer.Write(buf)
}

func readInt16(reader io.Reader) (int16, error) {
	buf := make([]byte, 2)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToInt16(buf), err
}

func writeInt8(writer io.Writer, val int8) (int, error) {
	typ := []byte{TypeInt8}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeInt8Blob(writer, val)
	return wrote + 1, err
}

func writeInt8Blob(writer io.Writer, val int8) (int, error) {
	buf := []byte{byte(val)}
	return writer.Write(buf)
}

func readInt8(reader io.Reader) (int8, error) {
	buf := make([]byte, 1)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return int8(buf[0]), err
}

func writeFloat64(writer io.Writer, val float64) (int, error) {
	typ := []byte{TypeDouble}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeFloat64Blob(writer, val)
	return wrote + 1, err
}

func writeFloat64Blob(writer io.Writer, val float64) (int, error) {
	return writer.Write(float64ToBytes(val))
}

func readFloat64(reader io.Reader) (float64, error) {
	buf := make([]byte, 8)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToFloat64(buf), err
}

func writeBool(writer io.Writer, val bool) (int, error) {
	typ := []byte{TypeBool}
	_, err := writer.Write(typ)
	if err != nil {
		return 0, err
	}

	wrote, err := writeBoolBlob(writer, val)
	return wrote + 1, err
}

func writeBoolBlob(writer io.Writer, val bool) (int, error) {
	buf := []byte{0}
	if val == true {
		buf[0] = 1
	}

	return writer.Write(buf)
}

func readBool(reader io.Reader) (bool, error) {
	buf := make([]byte, 1)
	n, err := io.ReadFull(reader, buf)
	err = checkEof(err, n, len(buf))

	if err != nil && err != io.EOF {
		return false, err
	}

	res := false
	if buf[0] == 1 {
		res = true
	}

	return res, err
}

func writeName(writer io.Writer, blob []byte) (int, error) {
	size, err := writer.Write([]byte{byte(len(blob))})
	if err != nil {
		return size, err
	}

	wrote, err := writer.Write(blob)
	return wrote + size, err
}

func readName(reader io.Reader) ([]byte, error) {
	size := []byte{0}
	_, err := reader.Read(size)

	if err == io.EOF {
		return nil, ErrUnexpectedEof
	}

	if err != nil {
		return nil, err
	}

	name := make([]byte, size[0])
	n, err := io.ReadFull(reader, name)
	err = checkEof(err, n, len(name))

	if err != nil && err != io.EOF {
		return nil, err
	}

	return name, err
}

func writeBinaryString(writer io.Writer, val []byte) (int, error) {
	_, err := writer.Write([]byte{TypeBinaryString})
	if err != nil {
		return 0, err
	}

	size, err := packVarint(writer, uint64(len(val)))
	if err != nil {
		return size, err
	}

	wrote, err := writer.Write(val)
	return wrote + size + 1, err
}

func readBinaryString(reader io.Reader) ([]byte, error) {
	size, err := unpackVarint(reader)
	if err == io.EOF {
		return nil, ErrUnexpectedEof
	}

	if err != nil {
		return nil, err
	}

	buf := make([]byte, size)
	n, err := io.ReadFull(reader, buf)

	err = checkEof(err, n, len(buf))
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf, err
}

func writeBlob(writer io.Writer, val []byte) (int, error) {
	return writer.Write(val)
}

func writeObjectTag(writer io.Writer) (int, error) {
	return writer.Write([]byte{TypeObject})
}
