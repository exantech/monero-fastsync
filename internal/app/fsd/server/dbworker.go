package server

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/moneroutil"
)

type DbWorker interface {
	GetBlocksAbove(startHeight uint64, maxCount int) ([]PreparsedBlock, error)
	GetBlockEntry(height uint64) (BlockEntry, error)
	GetChainIntersection(chain []moneroutil.Hash) (utils.HeightInfo, error)
	GetWalletBlocks(walletId uint32, startHeight uint64, maxBlocks int) ([]PreSerializedBlock, error)
	GetWalletOutputs(walletId uint32) ([]OutputHeight, error)
	SaveWalletBlocks(walletId uint32, blocks []moneroutil.Hash, outputs []OutputHeight) error
	SaveWalletProgress(walletId uint32, hash moneroutil.Hash) error
	GetTopScannedHeightInfo(walletId uint32) (utils.HeightInfo, error)
	GetOrCreateKeyProgress(account utils.AccountInfo) (utils.WalletEntry, error)
	GetTopBlockHeight() (uint64, error)
}

type WalletsDb struct {
	db *sql.DB
}

func NewDbWorker(settings utils.DbSettings) (DbWorker, error) {
	db, err := utils.NewDb(settings)
	if err != nil {
		return nil, err
	}

	return &WalletsDb{
		db: db,
	}, nil
}

type BlockEntry struct {
	Height uint64
	Hash   moneroutil.Hash
	Header []byte
}

type PreparsedBlock struct {
	BlockEntry
	Txs []PreparsedTx
}

type PreparsedTx struct {
	Hash          moneroutil.Hash
	Blob          []byte
	OutputKeys    []moneroutil.Key
	OutputIndices []uint64
	UsedInputs    []uint64
}

type PreSerializedBlock struct {
	Height uint64
	Hash   moneroutil.Hash
	Header []byte
	Txs    []ExtSerializedTx
}

type ExtSerializedTx struct {
	Hash          moneroutil.Hash
	Blob          []byte
	OutputIndices []uint64
}

type OutputHeight struct {
	OutputIndex uint64
	Height      uint64
}

func toStringList(hashes []moneroutil.Hash) string {
	b := bytes.NewBufferString("")
	for i, h := range hashes {
		if i != 0 {
			b.WriteString(", '")
			b.WriteString(h.String())
			b.WriteString("'")
		} else {
			b.WriteString("'")
			b.WriteString(h.String())
			b.WriteString("'")
		}
	}

	return b.String()
}

func (w *WalletsDb) GetChainIntersection(chain []moneroutil.Hash) (utils.HeightInfo, error) {
	row := w.db.QueryRow(fmt.Sprintf(`SELECT height, hash FROM blocks
		WHERE hash IN (%s)
		ORDER BY height DESC
		LIMIT 1`, toStringList(chain)))

	hi := utils.HeightInfo{}
	var hs string
	if err := row.Scan(&hi.Height, &hs); err != nil {
		return hi, err
	}

	var err error
	hi.Hash, err = moneroutil.HexToHash(hs)
	return hi, err
}

func (w *WalletsDb) GetTopBlockHeight() (uint64, error) {
	var height uint64
	if err := w.db.QueryRow(`SELECT height FROM blocks ORDER BY height DESC LIMIT 1`).Scan(&height); err != nil {
		logging.Log.Errorf("Failed to get top block height: %s", err.Error())
		return 0, err
	}

	return height, nil
}

func (w *WalletsDb) GetBlocksAbove(startHeight uint64, maxCount int) ([]PreparsedBlock, error) {
	rows, err := w.db.Query(
		`SELECT b.height, b.hash, b.header, t.hash, t.blob, t.output_keys, 
					t.output_indices, t.used_inputs
			  FROM transactions t
			  LEFT JOIN blocks b ON t.block_height = b.height
			  WHERE b.height >= $1 AND b.height < $2
			  ORDER BY b.height, t.index_in_block ASC`, startHeight, startHeight+uint64(maxCount))

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	blocks := make([]PreparsedBlock, 0, maxCount)
	for rows.Next() {
		var height uint64
		var blockHash string
		var blockHeader []byte
		var txHash string
		var txBlob []byte
		var outputKeys []string
		var outputIndices []int64 // libpq doesn't support reading of []uint64
		var usedInputs []int64    // libpq doesn't support reading of []uint64

		err = rows.Scan(
			&height,
			&blockHash,
			&blockHeader,
			&txHash,
			&txBlob,
			pq.Array(&outputKeys),
			pq.Array(&outputIndices),
			pq.Array(&usedInputs))

		if err != nil {
			logging.Log.Errorf("Failed to scan results on scanning blocks: %s", err.Error())
			return nil, err
		}

		if len(blocks) == 0 || blocks[len(blocks)-1].Height != height {
			h, err := moneroutil.HexToHash(blockHash)
			if err != nil {
				logging.Log.Errorf("Failed to decode block hash (%s) from DB: %s", blockHash, err.Error())
				return nil, err
			}

			block := PreparsedBlock{
				BlockEntry: BlockEntry{
					Height: height,
					Header: blockHeader,
					Hash:   h,
				},
				Txs: []PreparsedTx{},
			}

			blocks = append(blocks, block)
		}

		h, err := moneroutil.HexToHash(txHash)
		if err != nil {
			logging.Log.Errorf("Failed to decode transaction hash (%s) from DB: %s", txHash, err.Error())
		}

		keys, err := convertStringsToKeys(outputKeys)
		if err != nil {
			logging.Log.Errorf("Failed to decode output keys for transaction %s from DB: %s", txHash, err.Error())
		}

		tx := PreparsedTx{
			Hash:          h,
			Blob:          txBlob,
			OutputKeys:    keys,
			OutputIndices: convertInts64toUints(outputIndices),
			UsedInputs:    convertInts64toUints(usedInputs),
		}

		blocks[len(blocks)-1].Txs = append(blocks[len(blocks)-1].Txs, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (w *WalletsDb) GetBlockEntry(height uint64) (BlockEntry, error) {
	be := BlockEntry{}

	row := w.db.QueryRow(
		`SELECT b.height, b.hash, b.header
			  FROM blocks b
			  WHERE b.height = $1`, height)

	var hash string
	if err := row.Scan(&be.Height, &hash, &be.Header); err != nil {
		logging.Log.Errorf("Failed to get block entry at height %d: %s", height, err.Error())
		return be, err
	}

	var err error
	be.Hash, err = moneroutil.HexToHash(hash)
	if err != nil {
		logging.Log.Errorf("Failed to parse block hash string from db (%s): %s", hash, err.Error())
		return be, err
	}

	return be, nil
}

func (w *WalletsDb) GetWalletBlocks(walletId uint32, startHeight uint64, maxBlocks int) ([]PreSerializedBlock, error) {
	rows, err := w.db.Query(
		`SELECT wb.wallet_id, b.height, b.hash, b.header, t.hash, t.blob, t.output_indices
FROM blocks b
LEFT JOIN transactions t ON t.block_height = b.height
LEFT JOIN wallets_blocks wb ON wb.block_id = b.id AND wb.wallet_id = $3
WHERE b.height >= $1 AND b.height < $2
ORDER BY b.height, t.index_in_block ASC`, startHeight, startHeight+uint64(maxBlocks), walletId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	blocks := make([]PreSerializedBlock, 0, maxBlocks)
	for rows.Next() {
		var wId sql.NullInt64
		var height uint64
		var blockHash string
		var blockHeader []byte
		var txHash sql.NullString
		var txBlob []byte
		var outputIndices []int64

		err = rows.Scan(
			&wId,
			&height,
			&blockHash,
			&blockHeader,
			&txHash,
			&txBlob,
			pq.Array(&outputIndices))

		if err != nil {
			logging.Log.Errorf("Failed to scan results on scanning wallet blocks: %s", err.Error())
			return nil, err
		}

		if len(blocks) == 0 || blocks[len(blocks)-1].Height != height {
			h, err := moneroutil.HexToHash(blockHash)
			if err != nil {
				logging.Log.Errorf("Failed to decode block hash (%s) from DB: %s", blockHash, err.Error())
				return nil, err
			}

			header := blockHeader
			if !wId.Valid {
				header = nil
			}

			block := PreSerializedBlock{
				Height: height,
				Header: header,
				Hash:   h,
				Txs:    []ExtSerializedTx{},
			}

			blocks = append(blocks, block)
		}

		if !txHash.Valid {
			// coinbase transaction
			continue
		}

		h, err := moneroutil.HexToHash(txHash.String)
		if err != nil {
			logging.Log.Errorf("Failed to decode transaction hash (%s) from DB: %s", txHash, err.Error())
		}

		if !wId.Valid {
			continue
		}

		tx := ExtSerializedTx{
			Hash:          h,
			Blob:          txBlob,
			OutputIndices: convertInts64toUints(outputIndices),
		}

		blocks[len(blocks)-1].Txs = append(blocks[len(blocks)-1].Txs, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (w *WalletsDb) GetWalletOutputs(walletId uint32) ([]OutputHeight, error) {
	rows, err := w.db.Query(`SELECT output, block_height FROM wallets_outputs WHERE wallet_id = $1`, walletId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	outputs := make([]OutputHeight, 0, 200)
	for rows.Next() {
		var height uint64
		var output uint64

		err = rows.Scan(
			&output,
			&height)
		if err != nil {
			logging.Log.Errorf("Failed to scan results on scanning wallet's outputs: %s", err.Error())
			return nil, err
		}

		outputs = append(outputs, OutputHeight{output, height})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return outputs, nil
}

func (w *WalletsDb) SaveWalletBlocks(walletId uint32, blocks []moneroutil.Hash, outputs []OutputHeight) error {
	hashStrs := toStringList(blocks)

	tx, err := w.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		logging.Log.Errorf("Failed to begin transaction for saving wallet outputs: %s", err.Error())
		return err
	}

	defer tx.Rollback()

	q := fmt.Sprintf("INSERT INTO wallets_blocks (wallet_id, block_id) (SELECT %d, id FROM blocks WHERE hash IN (%s))",
		walletId, hashStrs)

	ir, err := tx.Exec(q)
	if err != nil {
		logging.Log.Errorf("Failed to insert wallet's blocks: %s", err.Error())
		return err
	}

	rows, err := ir.RowsAffected()
	if err != nil {
		logging.Log.Errorf("Failed to get affected rows count: %s", err.Error())
		return err
	}

	logging.Log.Debugf("Inserted %d wallet's blocks", rows)

	ostmt, err := tx.PrepareContext(context.Background(), "INSERT INTO wallets_outputs (wallet_id, output, block_height) VALUES ($1, $2, $3)")
	if err != nil {
		logging.Log.Errorf("Failed to prepare statement for saving wallet outputs: %s", err.Error())
		return err
	}

	defer ostmt.Close()

	for _, o := range outputs {
		_, err = ostmt.Exec(walletId, o.OutputIndex, o.Height)
		if err != nil {
			logging.Log.Errorf("Couldn't insert output into db: %s", err.Error())
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logging.Log.Debugf("Saved %d outputs for wallet %d", len(outputs), walletId)
	return nil
}

func (w *WalletsDb) SaveWalletProgress(walletId uint32, hash moneroutil.Hash) error {
	tx, err := w.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		logging.Log.Errorf("Failed to begin transaction for saving wallet's progress: %s", err.Error())
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`WITH last_block_id AS (SELECT id FROM blocks WHERE hash = $1)
UPDATE wallets
SET last_checked_block_id = last_block_id.id
FROM last_block_id
WHERE wallets.id = $2`, hash.String(), walletId)
	if err != nil {
		logging.Log.Errorf("Failed to update wallet's progress: %s", err.Error())
		return err
	}

	if err = tx.Commit(); err != nil {
		logging.Log.Errorf("Failed to commit wallet's progress: %s", err.Error())
		return err
	}

	return nil
}

func (w *WalletsDb) GetTopScannedHeightInfo(walletId uint32) (utils.HeightInfo, error) {
	res := utils.HeightInfo{}

	r := w.db.QueryRow(`SELECT b.height, b.hash
							FROM wallets
							LEFT JOIN blocks b on wallets.last_checked_block_id = b.id
							WHERE wallets.id = $1`, walletId)

	var hash string
	if err := r.Scan(&res.Height, &hash); err != nil {
		logging.Log.Errorf("Failed to scan get top scanned height info: %s", hash)
		return res, err
	}

	var err error
	res.Hash, err = moneroutil.HexToHash(hash)
	if err != nil {
		logging.Log.Errorf("Failed to parse hash hex while getting top scanned height info: %s", hash)
		return res, err
	}

	return res, nil
}

func (w *WalletsDb) GetOrCreateKeyProgress(account utils.AccountInfo) (utils.WalletEntry, error) {
	res := utils.WalletEntry{}

	tx, err := w.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		logging.Log.Errorf("Failed to begin transaction for getting wallet's progress: %s", err.Error())
		return res, err
	}

	defer tx.Rollback()

	r := tx.QueryRow(`SELECT w.id, b.height FROM wallets w
							LEFT JOIN blocks b ON w.last_checked_block_id = b.id
							WHERE secret_view_key = $1 AND public_spend_key = $2`,
		account.Keys.ViewSecretKey.String(), account.Keys.SpendPublicKey.String())

	err = r.Scan(&res.Id, &res.ScannedHeight)
	if err != nil && err != sql.ErrNoRows {
		logging.Log.Errorf("Failed to query wallet (%s, %s): %s",
			account.Keys.ViewSecretKey.String(), account.Keys.SpendPublicKey.String(), err.Error())
		return res, err
	}

	if err != sql.ErrNoRows {
		res.Keys = account.Keys
		tx.Commit()
		return res, nil
	}

	row := tx.QueryRow(`INSERT INTO wallets (secret_view_key, public_spend_key, created_at, last_checked_block_id)
					(SELECT $1, $2, $3, id FROM blocks WHERE height = $3 limit 1) RETURNING wallets.id`,
		account.Keys.ViewSecretKey.String(), account.Keys.SpendPublicKey.String(), account.CreatedAt)

	var id uint32
	if err = row.Scan(&id); err != nil {
		logging.Log.Errorf("Couldn't insert wallet: %s", err.Error())
	}

	res.ScannedHeight = account.CreatedAt
	res.Keys = account.Keys
	res.Id = uint32(id)

	tx.Commit()

	return res, nil
}

func convertStringsToKeys(strs []string) ([]moneroutil.Key, error) {
	keys := make([]moneroutil.Key, 0, len(strs))
	for _, s := range strs {
		k, err := moneroutil.HexToKey(s)
		if err != nil {
			logging.Log.Errorf("Failed to decode key %s, error: %s", s, err.Error())
			return nil, err
		}

		keys = append(keys, k)
	}

	return keys, nil
}

func convertInts64toUints(ints []int64) []uint64 {
	uints := make([]uint64, 0, len(ints))
	for _, i := range ints {
		uints = append(uints, uint64(i))
	}

	return uints
}
