package rpc

import (
	"bytes"
	"errors"

	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/moneroproto"
	"github.com/exantech/moneroutil"
)

type SupportedVersionsResponse struct {
	Versions []uint32 `monerobinkv:"supported_versions"`
}

type GetMyBlocksRequest struct {
	Version uint32            `monerobinkv:"version"`
	Params  WalletChainInfoV1 `monerobinkv:"params"`
}

type WalletChainInfoV1 struct {
	Keys       []WalletKeysInfo `monerobinkv:"keys"`
	ShortChain [][]byte         `monerobinkv:"short_chain"`
}

type WalletKeysInfo struct {
	KeyPair   [][]byte `monerobinkv:"keys"`
	CreatedAt uint64   `monerobinkv:"created_at"`
}

func (w *WalletKeysInfo) GetWalletKeys() (utils.WalletKeys, error) {
	res := utils.WalletKeys{}
	if len(w.KeyPair) != 2 {
		return res, errors.New("needed exactly 2 keys")
	}

	var err error
	r := bytes.NewReader(w.KeyPair[0])
	res.ViewSecretKey, err = moneroutil.ParseKey(r)
	if err != nil {
		return res, err
	}

	r = bytes.NewReader(w.KeyPair[1])
	res.SpendPublicKey, err = moneroutil.ParseKey(r)
	return res, err
}

func (w *WalletKeysInfo) SetWalletKeys(keys utils.WalletKeys) {
	w.KeyPair = make([][]byte, 2)

	w.KeyPair[0] = keys.ViewSecretKey.Serialize()
	w.KeyPair[1] = keys.SpendPublicKey.Serialize()
}

func (w *WalletChainInfoV1) GetShortChain() ([]moneroutil.Hash, error) {
	chain := make([]moneroutil.Hash, 0, len(w.ShortChain))
	for _, hb := range w.ShortChain {
		r := bytes.NewReader(hb)
		h, err := moneroutil.ParseHash(r)
		if err != nil {
			return nil, err
		}

		chain = append(chain, h)
	}

	return chain, nil
}

func (w *WalletChainInfoV1) SetShortChain(chain []moneroutil.Hash) {
	w.ShortChain = make([][]byte, len(chain))
	for i, h := range chain {
		w.ShortChain[i] = h.Serialize()
	}
}

type GetMyBlocksResponse struct {
	Status []byte             `monerobinkv:"status"`
	Result WalletBlocksResult `monerobinkv:"result"`
}

func (g *GetMyBlocksResponse) SetStatus(status string) {
	g.Status = []byte(status)
}

type WalletBlocksResult struct {
	StartHeight uint64            `monerobinkv:"start_height"`
	TotalHeight uint64            `monerobinkv:"total_height"`
	Blocks      []WalletBlockInfo `monerobinkv:"blocks"`
}

type WalletBlockInfo struct {
	Hash          []byte                         `monerobinkv:"hash"`
	Timestamp     uint64                         `monerobinkv:"timestamp"`
	Bce           moneroproto.BlockCompleteEntry `monerobinkv:"block"`
	OutputIndices moneroproto.BlockOutputIndices `monerobinkv:"output_indices"`
}

func (w *WalletBlockInfo) SetOutputIndices(outs [][]uint64) {
	w.OutputIndices.Indices = make([]moneroproto.TxOutputIndices, len(outs))

	for i, t := range outs {
		w.OutputIndices.Indices[i].Indices = t
	}
}
