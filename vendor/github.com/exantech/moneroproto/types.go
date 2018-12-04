package moneroproto

import (
	"encoding/hex"
	"errors"

	"github.com/exantech/moneroutil"
)

var (
	ErrLengthMismatch = errors.New("length mismatch")
)

func NewHashFromBytes(bytes []byte) *moneroutil.Hash {
	if len(bytes) != moneroutil.HashLength {
		panic("[]bytes length must be 32 bytes")
	}

	var hash moneroutil.Hash
	copy(hash[:], bytes)
	return &hash
}

func NewHashFromHexStr(str []byte) (error, *moneroutil.Hash) {
	if hex.DecodedLen(len(str)) != moneroutil.HashLength {
		return ErrLengthMismatch, nil
	}

	bytes := make([]byte, moneroutil.HashLength)
	_, err := hex.Decode(bytes, str)
	if err != nil {
		return err, nil
	}

	return nil, NewHashFromBytes(bytes)
}

//TODO: make hashes []*moneroutil.Hash
func HashesToByteSlice(hashes []moneroutil.Hash) []byte {
	res := make([]byte, 0, len(hashes)*moneroutil.HashLength)
	for _, hash := range hashes {
		res = append(res, hash.Serialize()...)
	}

	return res
}

//TODO: make hashes []*moneroutil.Hash
func ByteSliceToHashes(hashes []byte) (error, []moneroutil.Hash) {
	if len(hashes)%moneroutil.HashLength != 0 {
		panic("corrupted hashes slice")
	}

	hashesCount := len(hashes) / moneroutil.HashLength
	res := make([]moneroutil.Hash, 0, hashesCount)
	for i := 0; i < hashesCount; i++ {
		var hash moneroutil.Hash
		copy(hash[:], hashes[i*moneroutil.HashLength:(i+1)*moneroutil.HashLength])
		res = append(res, hash)
	}

	return nil, res
}
