package server

import (
	"errors"

	"github.com/exantech/monero-fastsync/internal/app/fsd/rpc"
	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

var (
	ErrRequestError  = errors.New("request error")
	ErrInternalError = errors.New("internal error")
)

type BlocksHandler struct {
	dbWorker DbWorker
	scanner  Scanner
	queue    *jobsQueue
}

func NewBlocksHandler(db DbWorker, queue *jobsQueue) *BlocksHandler {
	return &BlocksHandler{
		dbWorker: db,
		queue:    queue,
	}
}

func (b *BlocksHandler) HandleGetBlocks(req *rpc.WalletChainInfoV1) (*rpc.WalletBlocksResult, error) {
	accounts, err := accountsInfoFromWalletKeysInfo(req.Keys)
	if err != nil {
		logging.Log.Error("Failed to parse wallet keys: %s", err.Error())
		return nil, ErrRequestError
	}

	if len(accounts) == 0 {
		logging.Log.Error("Empty keys")
		return nil, ErrRequestError
	}

	if len(accounts) > 1 {
		logging.Log.Error("More than one key is not supported yet")
		return nil, ErrRequestError
	}

	chain, err := req.GetShortChain()
	if err != nil {
		logging.Log.Error("Failed to parse short chain: %s", err.Error())
		return nil, ErrRequestError
	}

	if len(chain) == 0 {
		logging.Log.Error("Empty short chain")
		return nil, ErrRequestError
	}

	common, err := b.dbWorker.GetChainIntersection(chain)
	if err != nil {
		logging.Log.Errorf("Failed to get common block: %s", err.Error())
		return nil, ErrInternalError
	}

	progress, err := b.dbWorker.GetOrCreateKeyProgress(accounts[0])
	if err != nil {
		logging.Log.Errorf("Failed to get wallets progress: %s", err.Error())
		return nil, ErrInternalError
	}

	listener := b.queue.AddJob(progress, common.Height)

	blocks, err := listener.Wait()
	if err != nil {
		logging.Log.Errorf("Failed to get wallet's blocks: %s", err.Error())
		return nil, ErrInternalError
	}

	logging.Log.Infof("Processed %d blocks", len(blocks))

	topHeight, err := b.dbWorker.GetTopBlockHeight()
	if err != nil {
		logging.Log.Errorf("Error while getting top block height: %s", err.Error())
		return nil, ErrInternalError
	}

	res := &rpc.WalletBlocksResult{
		StartHeight: common.Height,
		TotalHeight: topHeight,
		Blocks:      make([]rpc.WalletBlockInfo, len(blocks)),
	}

	for i := range blocks {
		res.Blocks[i].Hash = blocks[i].Hash.Serialize()
		res.Blocks[i].SetOutputIndices(blocks[i].OutputIndices)
		res.Blocks[i].Timestamp = blocks[i].Timestamp
		if blocks[i].Bce != nil {
			res.Blocks[i].Bce = *blocks[i].Bce
		}
	}

	return res, nil
}

func accountsInfoFromWalletKeysInfo(ws []rpc.WalletKeysInfo) ([]utils.AccountInfo, error) {
	res := make([]utils.AccountInfo, 0, len(ws))

	var err error
	for _, w := range ws {
		a := utils.AccountInfo{}
		a.Keys, err = w.GetWalletKeys()
		if err != nil {
			return nil, err
		}

		a.CreatedAt = w.CreatedAt
		res = append(res, a)
	}

	return res, nil
}
