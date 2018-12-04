package moneroproto

const (
	TypeInt64        byte = 0x01
	TypeInt32        byte = 0x02
	TypeInt16        byte = 0x03
	TypeInt8         byte = 0x04
	TypeUint64       byte = 0x05
	TypeUint32       byte = 0x06
	TypeUint16       byte = 0x07
	TypeUint8        byte = 0x08
	TypeDouble       byte = 0x09
	TypeBinaryString byte = 0x0a
	TypeBool         byte = 0x0b
	TypeObject       byte = 0x0c
	TypeArray        byte = 0x0d

	FlagArray byte = 0x80
)

const (
	MarkMask  byte = 0x03
	MarkByte  byte = 0x00
	MarkWord  byte = 0x01
	MarkDWord byte = 0x02
	MarkInt64 byte = 0x03
)

var (
	MessagePreamble = []byte{0x01, 0x11, 0x01, 0x01, 0x01, 0x01, 0x02, 0x01, 0x01}
)
