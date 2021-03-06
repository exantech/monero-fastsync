package utils

import (
	"errors"

	"github.com/exantech/moneroutil"
)

var (
	ErrInterrupted = errors.New("interrupted")
)

type HeightInfo struct {
	Height uint64
	Hash   moneroutil.Hash
}

type WalletKeys struct {
	ViewSecretKey  moneroutil.Key
	SpendPublicKey moneroutil.Key
}

type AccountInfo struct {
	Keys      WalletKeys
	CreatedAt uint64
}

type WalletEntry struct {
	Id            uint32
	Keys          WalletKeys
	ScannedHeight uint64
}

func MinUint64(a, b uint64) uint64 {
	min := a
	if b < min {
		min = b
	}

	return min
}

func MinInt(a, b int) int {
	min := a
	if b < min {
		min = b
	}

	return min
}
