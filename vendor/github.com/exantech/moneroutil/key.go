package moneroutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

const (
	KeyLength = 32
)

// Key can be a Scalar or a Point
type Key [KeyLength]byte

func (p *Key) FromBytes(b [KeyLength]byte) {
	*p = b
}

func (p *Key) ToBytes() (result [KeyLength]byte) {
	result = [KeyLength]byte(*p)
	return
}

func (p *Key) PubKey() (pubKey *Key) {
	point := new(ExtendedGroupElement)
	GeScalarMultBase(point, p)
	pubKey = new(Key)
	point.ToBytes(pubKey)
	return
}

// Creates a point on the Edwards Curve by hashing the key
func (p *Key) HashToEC() (result *ExtendedGroupElement) {
	result = new(ExtendedGroupElement)
	var p1 ProjectiveGroupElement
	var p2 CompletedGroupElement
	h := Key(Keccak256(p[:]))
	p1.FromBytes(&h)
	GeMul8(&p2, &p1)
	p2.ToExtended(result)
	return
}

func RandomScalar() (result *Key) {
	result = new(Key)
	var reduceFrom [KeyLength * 2]byte
	tmp := make([]byte, KeyLength*2)
	rand.Read(tmp)
	copy(reduceFrom[:], tmp)
	ScReduce(result, &reduceFrom)
	return
}

func NewKeyPair() (privKey *Key, pubKey *Key) {
	privKey = RandomScalar()
	pubKey = privKey.PubKey()
	return
}

func ParseKey(buf io.Reader) (Key, error) {
	key := [KeyLength]byte{}
	_, err := io.ReadFull(buf, key[:])
	return key, err
}

func (k *Key) String() string {
	return hex.EncodeToString(k[:])
}

func HexToKey(h string) (Key, error) {
	result := Key{}
	if len(h) != KeyLength*2 {
		return result, errors.New("key hex string must be 64 bytes long")
	}

	byteSlice, err := hex.DecodeString(h)
	if err != nil {
		return result, err
	}

	copy(result[:], byteSlice)
	return result, nil
}

func KeyDerivation(a, B *Key) Key {
	point := new(ExtendedGroupElement)
	point.FromBytes(B)

	production := new(ProjectiveGroupElement)
	GeScalarMult(production, a, point)

	tmp2 := new(CompletedGroupElement)
	result := new(ExtendedGroupElement)
	var got Key
	GeMul8(tmp2, production)

	tmp2.ToExtended(result)
	result.ToBytes(&got)
	return got
}
