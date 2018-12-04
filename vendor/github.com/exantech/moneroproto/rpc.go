package moneroproto

import "github.com/exantech/moneroutil"

type GetHashesFastRequest struct {
	BlockIds    []byte `monerobinkv:"block_ids"`
	StartHeight uint64 `monerobinkv:"start_height"`
}

func (g *GetHashesFastRequest) SetHashes(hashes []moneroutil.Hash) {
	g.BlockIds = HashesToByteSlice(hashes)
}

func (g *GetHashesFastRequest) GetHashes() (error, []moneroutil.Hash) {
	return ByteSliceToHashes(g.BlockIds)
}

type GetHashesFastResponse struct {
	BlockIds      []byte `monerobinkv:"m_block_ids"`
	StartHeight   uint64 `monerobinkv:"start_height"`
	CurrentHeight uint64 `monerobinkv:"current_height"`
	Status        []byte `monerobinkv:"status"`
	Untrusted     bool   `monerobinkv:"untrusted"`
}

func (g *GetHashesFastResponse) SetHashes(hashes []moneroutil.Hash) {
	g.BlockIds = HashesToByteSlice(hashes)
}

func (g *GetHashesFastResponse) GetHashes() (error, []moneroutil.Hash) {
	return ByteSliceToHashes(g.BlockIds)
}

type GetBlocksFastRequest struct {
	BlockIds    []byte `monerobinkv:"block_ids"`
	StartHeight uint64 `monerobinkv:"start_height"`
	Prune       bool   `monerobinkv:"prune"`
	NoMinerTx   bool   `monerobinkv:"no_miner_tx"`
}

func (g *GetBlocksFastRequest) SetHashes(hashes []moneroutil.Hash) {
	g.BlockIds = HashesToByteSlice(hashes)
}

func (g *GetBlocksFastRequest) GetHashes() (error, []moneroutil.Hash) {
	return ByteSliceToHashes(g.BlockIds)
}

type BlockCompleteEntry struct {
	Block []byte   `monerobinkv:"block"`
	Txs   [][]byte `monerobinkv:"txs"`
}

type TxOutputIndices struct {
	Indices []uint64 `monerobinkv:"indices"`
}

type BlockOutputIndices struct {
	Indices []TxOutputIndices `monerobinkv:"indices"`
}

type GetBlocksFastResponse struct {
	Blocks        []BlockCompleteEntry `monerobinkv:"blocks"`
	StartHeight   uint64               `monerobinkv:"start_height"`
	CurrentHeight uint64               `monerobinkv:"current_height"`
	Status        []byte               `monerobinkv:"status"`
	OutputIndices []BlockOutputIndices `monerobinkv:"output_indices"`
	Untrusted     bool                 `monerobinkv:"untrusted"`
}
