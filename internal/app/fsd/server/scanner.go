package server

import (
	"bytes"
	"strconv"

	"github.com/exantech/moneroproto"
	"github.com/exantech/moneroutil"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/metrics"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type Scanner interface {
	GetBlocks(startHeight uint64, wallet utils.WalletEntry, maxBlocks int) ([]*WalletBlock, error)
}

type BlocksScanner struct {
	db DbWorker
}

type WalletBlock struct {
	Hash          moneroutil.Hash
	Timestamp     uint64
	Bce           *moneroproto.BlockCompleteEntry
	OutputIndices [][]uint64
}

func NewScanner(db DbWorker) *BlocksScanner {
	return &BlocksScanner{
		db: db,
	}
}

func (b *BlocksScanner) GetBlocks(startHeight uint64, wallet utils.WalletEntry, maxBlocks int) ([]*WalletBlock, error) {
	logging.Log.Debugf("Requested blocks from height %d, processed till %d", startHeight, wallet.ScannedHeight)

	if wallet.ScannedHeight >= startHeight {
		knownCount := wallet.ScannedHeight - startHeight + 1
		//inclusive from start height
		blocks, err := b.getProcessedBlocks(wallet.Id, startHeight, utils.MinInt(maxBlocks, int(knownCount)))
		if err != nil {
			logging.Log.Errorf("Failed to process job. Error on getting wallet's blocks: %s", err.Error())
			return nil, err
		}

		go metrics.Graphite().SimpleSend("fsd.blocks.cached", strconv.Itoa(len(blocks)))
		return blocks, nil
	}

	// the result must include start height block
	sr, err := b.scanWalletBlocks(wallet, startHeight, maxBlocks)
	if err != nil {
		logging.Log.Errorf("Failed to process job. Error on scanning wallet's blocks: %s", err.Error())
		return nil, err
	}

	if err = b.db.SaveWalletProgress(wallet.Id, sr.lastCheckedBlock); err != nil {
		logging.Log.Warningf("Failed save wallets progress: %s. Probably chain split happened, reverting progress", err.Error())
	}

	go metrics.Graphite().SimpleSend("fsd.blocks.scanned", strconv.Itoa(len(sr.blocks)))
	return sr.blocks, nil
}

//include from start height
func (b *BlocksScanner) getProcessedBlocks(walletId uint32, startHeight uint64, maxBlocks int) ([]*WalletBlock, error) {
	var blocks []PreSerializedBlock
	blocks, err := b.db.GetWalletBlocks(walletId, startHeight, maxBlocks) //inclusive from start height
	if err != nil {
		return nil, err
	}

	res := make([]*WalletBlock, 0, len(blocks))
	for _, b := range blocks {
		if len(b.Txs) == 0 {
			res = append(res, &WalletBlock{Hash: b.Hash})
		} else {
			wb, err := convertPreserializedToWalletBlock(b)
			if err != nil {
				logging.Log.Errorf("Failed to parse block on height %d: %s", b.Height, err.Error())
				return nil, err
			}
			res = append(res, wb)
		}
	}

	return res, nil
}

type scanResult struct {
	blocks           []*WalletBlock
	lastCheckedBlock moneroutil.Hash
}

func (b *BlocksScanner) scanWalletBlocks(wallet utils.WalletEntry, startHeight uint64, maxCount int) (*scanResult, error) {
	scanFrom := utils.MinUint64(wallet.ScannedHeight+1, startHeight)

	outs, err := b.db.GetWalletOutputs(wallet.Id)
	if err != nil {
		logging.Log.Errorf("Failed to get outputs from DB from height: %s", maxCount)
		return nil, err
	}

	logging.Log.Debugf("Wallet has %d outputs in db", len(outs))

	logging.Log.Debugf("Requesting blocks %d to process from height %d", maxCount, scanFrom)
	blocks, err := b.db.GetBlocksAbove(scanFrom, maxCount)
	if err != nil {
		logging.Log.Errorf("Failed to get blocks from DB from height %d: %s", scanFrom, maxCount)
		return nil, err
	}

	logging.Log.Debugf("Retrieved %d blocks", len(blocks))

	scanner := newTxScanner(wallet.Id, wallet.Keys, outs)

	var lastBlockHash moneroutil.Hash
	walletBlocks := make([]moneroutil.Hash, 0, maxCount)
	foundBlocks := make([]*WalletBlock, 0, maxCount)
	for _, block := range blocks {
		found := false
		for _, tx := range block.Txs {
			prefix, err := moneroutil.ParseTransactionPrefixBytes(tx.Blob)
			if err != nil {
				logging.Log.Errorf("Failed to parse transaction prefix for %s: %s", tx.Hash.String(), err.Error())
				return nil, err
			}

			r := bytes.NewReader(prefix.Extra)
			extra, err := moneroutil.ParseTransactionExtra(r)
			if err != nil {
				// this is not critical
				logging.Log.Warningf("Failed to parse transaction extra for %s: %s", tx.Hash.String(), err.Error())
				continue
			}

			if scanner.searchWalletOutputs(block.Height, extra.PubKeys, tx.OutputKeys, tx.OutputIndices) {
				found = true
			}

			if scanner.searchWalletMixins(tx.UsedInputs) {
				found = true
			}
		}

		lastBlockHash = block.Hash

		if found {
			walletBlocks = append(walletBlocks, block.Hash)
		}

		if found && block.Height >= startHeight {
			b, err := convertPreparsedToWalletBlock(block)
			if err != nil {
				return nil, err
			}

			foundBlocks = append(foundBlocks, b)
		} else if block.Height >= startHeight {
			foundBlocks = append(foundBlocks, &WalletBlock{Hash: block.Hash})
		}
	}

	if len(walletBlocks) != 0 {
		if err = b.db.SaveWalletBlocks(wallet.Id, walletBlocks, scanner.newOuts); err != nil {
			logging.Log.Errorf("Failed to save found outputs: %s", err.Error())
			return nil, err
		}
	}

	return &scanResult{
		blocks:           foundBlocks,
		lastCheckedBlock: lastBlockHash,
	}, nil
}

type txScanner struct {
	id      uint32
	wallet  utils.WalletKeys
	newOuts []OutputHeight
	outs    map[uint64]bool
}

func newTxScanner(walletId uint32, keys utils.WalletKeys, outs []OutputHeight) *txScanner {
	o := make(map[uint64]bool)
	for _, out := range outs {
		o[out.OutputIndex] = true
	}

	return &txScanner{
		id:      walletId,
		wallet:  keys,
		outs:    o,
		newOuts: []OutputHeight{},
	}
}

func (t *txScanner) searchWalletOutputs(height uint64, txPubKeys []moneroutil.Key, outputKeys []moneroutil.Key, globalIndices []uint64) bool {
	found := false
	for _, pubKey := range txPubKeys {
		derivation := moneroutil.KeyDerivation(&t.wallet.ViewSecretKey, &pubKey)

		for oi, outKey := range outputKeys {
			if outKey == calcOutputPubKey(derivation, oi, t.wallet.SpendPublicKey) {
				t.newOuts = append(t.newOuts, OutputHeight{globalIndices[oi], height})
				t.outs[globalIndices[oi]] = true

				found = true
			}
		}
	}

	return found
}

func (t *txScanner) searchWalletMixins(inputs []uint64) bool {
	for _, i := range inputs {
		_, ok := t.outs[i]
		if ok {
			return true
		}
	}

	return false
}

func calcOutputPubKey(derivation moneroutil.Key, index int, spendPublic moneroutil.Key) moneroutil.Key {
	buf := make([]byte, moneroutil.KeyLength)
	copy(buf, derivation[:])
	buf = append(buf, moneroutil.Uint64ToBytes(uint64(index))...)

	k := moneroutil.HashToScalar(buf)
	K := k.PubKey()

	P := moneroutil.Identity
	moneroutil.AddKeys(&P, K, &spendPublic)
	return P
}

func convertPreparsedToWalletBlock(block PreparsedBlock) (*WalletBlock, error) {
	res := &WalletBlock{}

	r := bytes.NewReader(block.Header)
	header, err := moneroutil.ParseBlockHeader(r)
	if err != nil {
		logging.Log.Errorf("Failed to convert PreparsedBlock to WalletBlock (hash: %s): %s", block.Hash.String(), err.Error())
		return res, err
	}

	miner, err := moneroutil.ParseTransactionBytes(block.Txs[0].Blob)
	if err != nil {
		logging.Log.Errorf("Failed to parse transaction (hash: %s): %s", block.Txs[0].Hash.String(), err.Error())
		return res, err
	}

	hashes := make([]moneroutil.Hash, 0, len(block.Txs)-1)
	for i := 1; i < len(block.Txs); i++ {
		hashes = append(hashes, block.Txs[i].Hash)
	}

	b := moneroutil.Block{
		BlockHeader: *header,
		MinerTx:     *miner,
		TxHashes:    hashes,
	}

	bce := moneroproto.BlockCompleteEntry{}
	bce.Block = serializeBlock(&b, block.Txs[0].Blob)

	res.OutputIndices = make([][]uint64, 0, len(block.Txs))
	res.OutputIndices = append(res.OutputIndices, block.Txs[0].OutputIndices)

	for i := 1; i < len(block.Txs); i++ {
		tx := block.Txs[i]
		bce.Txs = append(bce.Txs, tx.Blob)
		res.OutputIndices = append(res.OutputIndices, tx.OutputIndices)
	}

	res.Hash = block.Hash
	res.Bce = &bce
	res.Timestamp = header.TimeStamp

	return res, nil
}

func convertPreserializedToWalletBlock(block PreSerializedBlock) (*WalletBlock, error) {
	res := &WalletBlock{}

	if len(block.Txs) == 0 {
		return res, nil
	}

	r := bytes.NewReader(block.Header)
	header, err := moneroutil.ParseBlockHeader(r)
	if err != nil {
		logging.Log.Errorf("Failed to convert PreparsedBlock to WalletBlock (hash: %s): %s", block.Hash.String(), err.Error())
		return res, err
	}

	miner, err := moneroutil.ParseTransactionBytes(block.Txs[0].Blob)
	if err != nil {
		logging.Log.Errorf("Failed to parse transaction (hash: %s): %s", block.Txs[0].Hash.String(), err.Error())
		return res, err
	}

	hashes := make([]moneroutil.Hash, 0, len(block.Txs)-1)
	for i := 1; i < len(block.Txs); i++ {
		hashes = append(hashes, block.Txs[i].Hash)
	}

	b := moneroutil.Block{
		BlockHeader: *header,
		MinerTx:     *miner,
		TxHashes:    hashes,
	}

	bce := moneroproto.BlockCompleteEntry{}
	bce.Block = serializeBlock(&b, block.Txs[0].Blob)

	res.OutputIndices = make([][]uint64, 0, len(block.Txs))
	res.OutputIndices = append(res.OutputIndices, block.Txs[0].OutputIndices)

	for i := 1; i < len(block.Txs); i++ {
		tx := block.Txs[i]
		bce.Txs = append(bce.Txs, tx.Blob)
		res.OutputIndices = append(res.OutputIndices, tx.OutputIndices)
	}

	res.Hash = block.Hash
	res.Bce = &bce
	res.Timestamp = header.TimeStamp

	return res, nil
}

// TODO: make it part of moneroutil package!
func serializeBlock(b *moneroutil.Block, minerTx []byte) []byte {
	ser := make([]byte, 0)
	ser = append(ser, b.SerializeBlockHeader()...)
	ser = append(ser, minerTx...)
	ser = append(ser, moneroutil.Uint64ToBytes(uint64(len(b.TxHashes)))...)
	for _, h := range b.TxHashes {
		ser = append(ser, h.Serialize()...)
	}
	return ser
}
