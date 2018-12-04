package moneroutil

import (
	"bytes"
)

type Address struct {
	Network  int
	SpendKey Key
	ViewKey  Key
}

func (a *Address) Base58() (result string) {
	prefix := []byte{byte(a.Network)}
	checksum := GetChecksum(prefix, a.SpendKey[:], a.ViewKey[:])
	result = EncodeMoneroBase58(prefix, a.SpendKey[:], a.ViewKey[:], checksum[:])
	return
}

func NewAddress(address string) (result *Address, err string) {
	raw := DecodeMoneroBase58(address)
	if len(raw) != 69 {
		err = "Address is the wrong length"
		return
	}
	checksum := GetChecksum(raw[:65])
	if bytes.Compare(checksum[:], raw[65:]) != 0 {
		err = "Checksum does not validate"
		return
	}
	result = &Address{
		Network:  int(raw[0]),
	}
	copy(result.SpendKey[:], raw[1:33])
	copy(result.ViewKey[:], raw[33:65])
	return
}
