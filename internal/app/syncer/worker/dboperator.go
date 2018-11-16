package worker

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/exantech/moneroutil"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type DbOperator interface {
	GetShortChain() ([]utils.HeightInfo, error)
	SaveParsedBlocks(ctx context.Context, blocks []ParsedBlockInfo) error
	GetLastBlockHeight() (*uint64, error)
	TrimBlockchain(ctx context.Context, height uint64) error
	GetBlockHash(height uint64) (*moneroutil.Hash, error)
}

func NewDbOperator(settings utils.DbSettings) (DbOperator, error) {
	db, err := utils.NewDb(settings)
	if err != nil {
		return nil, err
	}

	return &PgOperator{
		db: db,
	}, nil
}

type ParsedBlockInfo struct {
	Height       uint64
	Hash         moneroutil.Hash
	Header       []byte
	Timestamp    uint32
	Transactions []ParsedTransactionInfo
}

type PgOperator struct {
	db *sql.DB
}

func (p *PgOperator) GetLastBlockHeight() (*uint64, error) {
	var height uint64
	err := p.db.QueryRow("SELECT height FROM blocks ORDER BY 1 DESC LIMIT 1").Scan(&height)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &height, err
}

func (p *PgOperator) GetShortChain() ([]utils.HeightInfo, error) {
	height, err := p.GetLastBlockHeight()
	if height == nil || err != nil {
		return []utils.HeightInfo{}, err
	}

	heights := calcShortChainHeights(*height)
	//workaround
	rows, err := p.db.Query(fmt.Sprintf("SELECT height, hash FROM blocks WHERE height in (%s) ORDER BY 1 DESC", uints64ToString(heights)))
	if err != nil {
		return []utils.HeightInfo{}, err
	}

	defer rows.Close()

	chain := make([]utils.HeightInfo, 0, 30)
	for rows.Next() {
		height := uint64(0)
		var hashHex string

		err = rows.Scan(&height, &hashHex)
		if err != nil {
			return []utils.HeightInfo{}, err
		}

		h, err := moneroutil.HexToHash(hashHex)
		if err != nil {
			logging.Log.Warningf("Couldn't parse hex string: %s", hashHex)
			return []utils.HeightInfo{}, err
		}

		chain = append(chain, utils.HeightInfo{height, h})
	}

	if err = rows.Err(); err != nil {
		logging.Log.Errorf("Couldn't fetch blocks from db: %s", err.Error())
		return []utils.HeightInfo{}, err
	}

	return chain, nil
}

func (p *PgOperator) SaveParsedBlocks(ctx context.Context, blocks []ParsedBlockInfo) error {
	logging.Log.Debug("Saving parsed blocks")

	if len(blocks) == 0 {
		logging.Log.Debug("No blocks to insert into db")
		return nil
	}

	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		logging.Log.Errorf("Couldn't begin transaction: %s", err.Error())
		return err
	}

	defer func() {
		tx.Rollback()
	}()

	logging.Log.Debug("Preparing insert blocks statement")
	blocksStmt, err := tx.PrepareContext(ctx, "INSERT INTO blocks (height, hash, header, timestamp) VALUES ($1, $2, $3, $4);")
	if err != nil {
		logging.Log.Error("Couldn't prepare insert blocks statement: %s", err.Error())
		return err
	}
	defer blocksStmt.Close()

	for _, block := range blocks {
		_, err = blocksStmt.Exec(block.Height, block.Hash.String(), block.Header, block.Timestamp)
		if err != nil {
			logging.Log.Errorf("Couldn't insert block into db: %s", err.Error())
			return err
		}
	}

	logging.Log.Debugf("Blocks inserted: %d, blocks: %s(%d)...%s(%d)", len(blocks),
		blocks[0].Hash.String(), blocks[0].Height, blocks[len(blocks)-1].Hash.String(), blocks[len(blocks)-1].Height)

	logging.Log.Debug("Preparing insert transactions statement")
	txsStmt, err := tx.PrepareContext(ctx, "INSERT INTO transactions (hash, blob, index_in_block, output_keys, output_indices, used_inputs, timestamp, block_height)"+
		" VALUES($1, $2, $3, $4, $5, $6, $7, $8)")

	if err != nil {
		logging.Log.Errorf("Couldn't prepare insert transactions statement: %s", err.Error())
		return err
	}
	defer txsStmt.Close()

	for _, block := range blocks {
		for idx, tr := range block.Transactions {
			keys := convertKeysToStringArray(tr.OutputKeys)
			_, err = txsStmt.Exec(tr.Hash.String(), tr.Blob, idx, pq.Array(keys), pq.Array(tr.OutputIndices), pq.Array(tr.UsedInInputs), tr.Timestamp, block.Height)
			if err != nil {
				logging.Log.Errorf("Couldn't insert transactions into db: %s", err.Error())
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		logging.Log.Errorf("Error on committing transaction: %s", err.Error())
		return err
	}

	return nil
}

func (p *PgOperator) TrimBlockchain(ctx context.Context, height uint64) error {
	logging.Log.Debugf("Trimming blockchain from height %d", height)

	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		logging.Log.Errorf("Couldn't begin transaction: %s", err.Error())
		return err
	}

	defer tx.Rollback()

	res, err := tx.Exec("DELETE FROM transactions WHERE block_height >= $1", height)
	if err != nil {
		logging.Log.Errorf("Couldn't trim 'transactions' table: %s", err.Error())
		return err
	}

	trimmedTxs, _ := res.RowsAffected()
	logging.Log.Debugf("Trimmed %d transactions", trimmedTxs)

	res, err = tx.Exec("DELETE FROM wallets_blocks USING blocks WHERE block_id = blocks.id AND blocks.height >= $1", height)
	if err != nil {
		logging.Log.Errorf("Couldn't trim 'wallets_blocks' table: %s", err.Error())
		return err
	}

	trimmedWbs, _ := res.RowsAffected()
	logging.Log.Debugf("Trimmed %d wallets_blocks", trimmedWbs)

	res, err = tx.Exec("DELETE FROM wallets_outputs WHERE block_height >= $1", height)
	if err != nil {
		logging.Log.Errorf("Couldn't trim 'wallets_outputs' table: %s", err.Error())
		return err
	}

	trimmedOuts, _ := res.RowsAffected()
	logging.Log.Debugf("Trimmed %d wallets_outputs", trimmedOuts)

	res, err = tx.Exec(`WITH last_block_id AS (SELECT id FROM blocks WHERE height = $1)
		UPDATE wallets
		SET last_checked_block_id = last_block_id.id
		FROM last_block_id
		WHERE last_checked_block_id > last_block_id.id`, height-1)
	if err != nil {
		logging.Log.Errorf("Couldn't update 'wallets' table: %s", err.Error())
		return err
	}

	updatedWs, _ := res.RowsAffected()
	logging.Log.Debugf("Updated %d wallets", updatedWs)

	res, err = tx.Exec("DELETE FROM blocks WHERE height >= $1", height)
	if err != nil {
		logging.Log.Errorf("Couldn't trim 'blocks' table: %s", err.Error())
		return err
	}

	trimmedBlocks, _ := res.RowsAffected()
	logging.Log.Debugf("Trimmed %d blocks", trimmedBlocks)

	if err = tx.Commit(); err != nil {
		logging.Log.Errorf("Failed to commit transaction: %s", err.Error())
	}

	logging.Log.Debug("Trimming transaction committed")
	return nil
}

func (p *PgOperator) GetBlockHash(height uint64) (*moneroutil.Hash, error) {
	logging.Log.Debugf("Getting block hash on height %d", height)

	row := p.db.QueryRow("SELECT hash FROM blocks WHERE height = $1", height)

	var hashHex string
	err := row.Scan(&hashHex)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		logging.Log.Errorf("Failed to get block hash on height %d: %s", height, err.Error())
		return nil, err
	}

	res, err := moneroutil.HexToHash(hashHex)
	if err != nil {
		logging.Log.Errorf("Failed to parse hash hex (%s) from db: %s", hashHex, err.Error())
		return nil, err
	}

	return &res, nil
}

func convertKeysToStringArray(keys []moneroutil.Key) []string {
	res := make([]string, 0, len(keys))

	for _, key := range keys {
		res = append(res, key.String())
	}

	return res
}

func calcShortChainHeights(height uint64) []uint64 {
	chain := make([]uint64, 0, 30)

	for i := uint64(0); i < height; {
		chain = append(chain, height-i)

		if i < 10 {
			i += 1
		} else {
			i *= 2
		}

		if i >= height {
			i = height
		}
	}

	if len(chain) == 0 || chain[len(chain)-1] != 0 {
		chain = append(chain, 0)
	}

	return chain
}

func uints64ToString(ns []uint64) string {
	var r string
	for i, n := range ns {
		if i == 0 {
			r = fmt.Sprintf("%d", n)
			continue
		}

		r = fmt.Sprintf("%s, %d", r, n)
	}

	return r
}
