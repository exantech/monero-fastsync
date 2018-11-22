package worker

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/exantech/moneroutil"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/monero-fastsync/pkg/genesis"
)

const (
	syncedPollInterval = 30 * time.Second
)

type Worker struct {
	db      DbOperator
	node    NodeFetcher
	genesis *genesis.GenesisBlockInfo
}

func NewWorker(db DbOperator, node NodeFetcher, genesisInfo *genesis.GenesisBlockInfo) *Worker {
	return &Worker{
		db:      db,
		node:    node,
		genesis: genesisInfo,
	}
}

func (w *Worker) RunSyncLoop(ctx context.Context) <-chan error {
	done := make(chan error)
	go func() {
		err := w.syncLoop(ctx)
		done <- err
	}()

	return done
}

type ParsedTransactionInfo struct {
	Hash          moneroutil.Hash
	Blob          []byte
	OutputKeys    []moneroutil.Key
	OutputIndices []uint64
	UsedInInputs  []uint64
	Timestamp     uint32
}

func (w *Worker) CheckGenesis(ctx context.Context, init bool) error {
	dbGenesis, err := w.db.GetBlockHash(0)
	if err != nil {
		return err
	}

	if init {
		if dbGenesis == nil {
			tx, err := moneroutil.ParseTransactionBytes(w.genesis.TxBlob)
			if err != nil {
				logging.Log.Errorf("Failed to parse genesis transaction: %s", err.Error())
				return err
			}

			t := ParsedTransactionInfo{
				Hash:          tx.GetHash(),
				Blob:          w.genesis.TxBlob,
				OutputKeys:    extractOutputKeysArray(tx.Vout),
				OutputIndices: []uint64{0},
				UsedInInputs:  inflateInputs(extractUsedInputs(tx.Vin)),
			}

			if err = w.db.SaveParsedBlocks(ctx, []ParsedBlockInfo{{
				0, w.genesis.Hash, w.genesis.Header, w.genesis.Timestamp, []ParsedTransactionInfo{t},
			}}); err != nil {
				logging.Log.Errorf("Failed to insert genesis block: %s", err.Error())
				return err
			}
		} else {
			return errors.New("DB already contains genesis block")
		}
	} else {
		if dbGenesis == nil {
			return errors.New("DB does not contain genesis block info, run with '-init' " +
				"flag to populate it with initial blockchain information")
		} else if *dbGenesis != w.genesis.Hash {
			logging.Log.Errorf("Genesis in DB: %s, but we got: %s", *dbGenesis, w.genesis.Hash.String())
			return errors.New("genesis block mismatch")
		}
	}

	return nil
}

func (w *Worker) syncLoop(ctx context.Context) error {
	synced := false

	for {
		if cancelled(ctx) {
			logging.Log.Info("Interrupting sync loop")
			return utils.ErrInterrupted
		}

		if synced {
			select {
			case <-ctx.Done():
				logging.Log.Info("Interrupting sync loop")
				return utils.ErrInterrupted
			case <-time.After(syncedPollInterval):
				synced = false
			}
		}

		logging.Log.Debug("Getting short chain")

		// higher heights at the beginning of the slice
		shortChain, err := w.db.GetShortChain()
		if err != nil {
			logging.Log.Errorf("Failed to get short chain: %s", err.Error())
			return err
		}

		if len(shortChain) == 0 {
			logging.Log.Error("DB is empty. Try running with '-init' option")
			return errors.New("empty DB")
		}

		lastHeight := shortChain[0].Height
		logging.Log.Debugf("Last known height: %d, hash: %s", lastHeight, shortChain[0].Hash.String())

		logging.Log.Debug("Requesting blocks from node")
		resp, err := w.node.GetBlocks(shortChain, 0)
		if err != nil {
			logging.Log.Errorf("Failed to fetch blocks from node: %s", err.Error())
			return err
		}

		logging.Log.Debugf("Fetched %d blocks", len(resp.Blocks))

		if resp.StartHeight != lastHeight {
			logging.Log.Infof("Blockchain reorganize needed. Last known height: %d, daemon start height: %d",
				lastHeight, resp.StartHeight)

			if err = w.db.TrimBlockchain(ctx, resp.StartHeight+1); err != nil {
				logging.Log.Errorf("Failed to trim blockchain: %s", err.Error())
				return err
			}

			lastHeight = resp.StartHeight
			shortChain = trimShortChain(shortChain, lastHeight)
			logging.Log.Infof("Blockchain trimmed. Top block now: %d, hash: %s", lastHeight, shortChain[0].Hash.String())
		}

		topBlock, err := moneroutil.ParseBlockBytes(resp.Blocks[0].Block)
		if err != nil {
			logging.Log.Errorf("Failed to parse first block: %s", err.Error())
			return err
		}

		// is it possible?
		if topBlock.GetHash() != shortChain[0].Hash {
			logging.Log.Errorf("daemon response: {first block: %s, start height: %d}, db: {top hash: %s, top height: %d}",
				topBlock.GetHash().String(), resp.StartHeight, shortChain[0].Hash.String(), shortChain[0].Height)

			return errors.New("topBlock.GetHash() != shortChain[0].Hash. Shouldn't happen")
		}

		if len(resp.Blocks) == 1 {
			logging.Log.Debug("Blockchain is synchronized")
			synced = true
			continue
		}

		readyBlocks := make([]ParsedBlockInfo, 0, len(resp.Blocks))
		for blockIdx, bce := range resp.Blocks {
			if blockIdx == 0 {
				continue
			}

			if cancelled(ctx) {
				logging.Log.Info("Interrupting sync loop")
				return utils.ErrInterrupted
			}

			block, err := moneroutil.ParseBlockBytes(bce.Block)
			if err != nil {
				logging.Log.Errorf("Failed to parse block: %s", err.Error())
				return err
			}

			blockInfo := transformBlock(lastHeight+uint64(blockIdx), block, resp.OutputIndices[blockIdx].Indices[0].Indices)

			for txIdx, txb := range bce.Txs {
				txPrefix, err := moneroutil.ParseTransactionPrefixBytes(txb)
				if err != nil {
					logging.Log.Errorf("Failed to parse transaction: %s, "+
						"block hash: %s, transaction index: %d, transaction blob: %s",
						err.Error(), block.GetHash().String(), txIdx+1, hex.EncodeToString(txb))
					return err
				}

				blockInfo.Transactions = append(blockInfo.Transactions, ParsedTransactionInfo{
					Hash:          block.TxHashes[txIdx],
					Blob:          txb,
					OutputKeys:    extractOutputKeysArray(txPrefix.Vout),
					OutputIndices: resp.OutputIndices[blockIdx].Indices[txIdx+1].Indices,
					UsedInInputs:  inflateInputs(extractUsedInputs(txPrefix.Vin)),
				})
			}

			readyBlocks = append(readyBlocks, blockInfo)
		}

		if cancelled(ctx) {
			logging.Log.Info("Interrupting sync loop")
			return utils.ErrInterrupted
		}

		err = w.db.SaveParsedBlocks(ctx, readyBlocks)
		if err != nil {
			logging.Log.Errorf("Failed to save parsed blocks: %s", err.Error())
			return err
		}
	}

	return nil
}

func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func transformBlock(height uint64, block *moneroutil.Block, indices []uint64) ParsedBlockInfo {
	return ParsedBlockInfo{
		Height:    height,
		Hash:      block.GetHash(),
		Header:    block.SerializeBlockHeader(),
		Timestamp: uint32(block.TimeStamp),
		Transactions: []ParsedTransactionInfo{{
			Hash:          block.MinerTx.GetHash(),
			Blob:          block.MinerTx.Serialize(),
			OutputKeys:    extractOutputKeysArray(block.MinerTx.Vout),
			OutputIndices: indices,
			UsedInInputs:  inflateInputs(extractUsedInputs(block.MinerTx.Vin)),
		}},
	}
}

func extractOutputKeysArray(outs []*moneroutil.TxOut) []moneroutil.Key {
	res := make([]moneroutil.Key, 0, len(outs))
	for _, out := range outs {
		res = append(res, out.Key)
	}

	return res
}

func extractUsedInputs(ins []moneroutil.TxInSerializer) []uint64 {
	res := make([]uint64, 0, len(ins)*11)
	for _, in := range ins {
		toKey, ok := in.(*moneroutil.TxInToKey)
		if !ok {
			// this is coinbase transaction
			continue
		}

		res = append(res, toKey.KeyOffsets...)
	}

	return res
}

func inflateInputs(deflated []uint64) []uint64 {
	inflated := make([]uint64, len(deflated))
	for i, x := range deflated {
		if i == 0 {
			inflated[i] = x
			continue
		}

		inflated[i] = inflated[i-1] + x
	}

	return inflated
}

func trimShortChain(shortChain []utils.HeightInfo, lastHeight uint64) []utils.HeightInfo {
	for i, hi := range shortChain {
		if hi.Height == lastHeight {
			return shortChain[i:]
		}
	}

	return shortChain
}
