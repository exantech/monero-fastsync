package main

import (
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/exantech/moneroproto"
	"github.com/exantech/moneroutil"

	"github.com/exantech/monero-fastsync/internal/app/fsd/rpc"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/monero-fastsync/pkg/genesis"
)

var (
	spend = "28184c08eb7dd8d365af926dd4e2ed2685688c207e9d3f7a9e2d4d5dc8aaa64e"
	view  = "1a7f0e754421317b633aabce2a98e41fa11afeb9c5be5037e727e51f3ef96600"

	url = "http://127.0.0.1:18081/fastsync.bin"
)

func main() {
	createdHash, _ := moneroutil.HexToHash("ab30e42809d6170dcba84d8991ae07311a9f0482097a88f361f6f85ca37deb14")
	var lastHash *moneroutil.Hash
	lastHash = &createdHash

	for {
		req := rpc.GetMyBlocksRequest{
			Version: 1,
		}

		viewKey, _ := moneroutil.HexToKey(view)
		spendKey, _ := moneroutil.HexToKey(spend)

		ki := rpc.WalletKeysInfo{}
		ki.CreatedAt = 110000
		ki.SetWalletKeys(utils.WalletKeys{viewKey, spendKey})

		if lastHash == nil {
			req.Params.SetShortChain([]moneroutil.Hash{genesis.GetGenesisBlockInfo("stagenet").Hash})
		} else {
			req.Params.SetShortChain([]moneroutil.Hash{genesis.GetGenesisBlockInfo("stagenet").Hash, *lastHash})
		}

		req.Params.Keys = []rpc.WalletKeysInfo{ki}

		buffer := bytes.Buffer{}
		moneroproto.Write(&buffer, req)

		start := time.Now()
		resp, err := http.Post(url, "application/octet-stream", &buffer)
		log.Printf("Request time: %s", time.Now().Sub(start))

		if err != nil {
			log.Fatalf("Failed to perform request: %s", err.Error())
		}

		bresp := rpc.GetMyBlocksResponse{}
		err = moneroproto.Read(resp.Body, &bresp)

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Response status: %s", string(bresp.Status))
		}

		if err != nil && err != io.EOF {
			log.Fatalf("Failed to deserialize response: %s", err.Error())
		}

		log.Printf("start height: %d", bresp.Result.StartHeight)
		log.Printf("total height: %d", bresp.Result.TotalHeight)
		log.Printf("blocks acquired: %d", len(bresp.Result.Blocks))

		if bresp.Result.StartHeight == bresp.Result.TotalHeight {
			log.Printf("Wallet synchronized")
			return
		}

		for i, b := range bresp.Result.Blocks {
			if len(b.Bce.Block) != 0 {
				log.Printf("My block height: %d, hash: %s", bresp.Result.StartHeight+uint64(i), hex.EncodeToString(b.Hash))
			}
		}

		if len(bresp.Result.Blocks) != 0 {
			r := bytes.NewReader(bresp.Result.Blocks[len(bresp.Result.Blocks)-1].Hash)
			h, err := moneroutil.ParseHash(r)
			if err != nil {
				log.Fatalf("Failed to parse hash: %s", hex.EncodeToString(bresp.Result.Blocks[len(bresp.Result.Blocks)-1].Hash))
			}

			lastHash = &h
		}
	}
}
