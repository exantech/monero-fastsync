package worker

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/exantech/moneroproto"
	"github.com/exantech/moneroutil"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type NodeFetcher interface {
	GetBlocks(shortChain []utils.HeightInfo, lastHeight uint64) (*moneroproto.GetBlocksFastResponse, error)
}

func NewNodeFetcher(nodeAddress string) (NodeFetcher, error) {
	return &RealNodeFetcher{
		nodeAddress,
	}, nil
}

type RealNodeFetcher struct {
	address string
}

//TODO: make cancellation
func (r *RealNodeFetcher) GetBlocks(shortChain []utils.HeightInfo, lastHeight uint64) (*moneroproto.GetBlocksFastResponse, error) {
	req := moneroproto.GetBlocksFastRequest{
		StartHeight: lastHeight,
	}

	req.SetHashes(extractHeightHashes(shortChain))

	buffer := bytes.Buffer{}
	err := moneroproto.Write(&buffer, req)
	if err != nil {
		return nil, err
	}

	logging.Log.Debug("Fetching blocks from node")
	resp, err := http.Post(r.address+"/getblocks.bin", "application/octet-stream", &buffer)
	if err != nil {
		logging.Log.Errorf("Failed to fetch blocks from node: %s", err.Error())
		return nil, err
	}

	if resp.StatusCode != 200 {
		logging.Log.Errorf("Node returned error code: %s. This may mean the node is not functional or uses wrong blockchain. ", resp.Status)
		return nil, errors.New("server error")
	}

	logging.Log.Debug("Parsing blocks...")
	//TODO: metrics
	var blocksResp moneroproto.GetBlocksFastResponse
	err = moneroproto.Read(resp.Body, &blocksResp)

	if err != nil && err != io.EOF {
		logging.Log.Errorf("Failed to parse blocks: %s", err.Error())
		return nil, err
	}

	if string(blocksResp.Status) != "OK" {
		logging.Log.Errorf("Server responded with error: %s", blocksResp.Status)
		return nil, errors.New("server error")
	}

	logging.Log.Debug("Blocks parsed successfully")
	return &blocksResp, nil
}

func extractHeightHashes(shortChain []utils.HeightInfo) []moneroutil.Hash {
	res := make([]moneroutil.Hash, 0, len(shortChain))
	for _, h := range shortChain {
		res = append(res, h.Hash)
	}

	return res
}
