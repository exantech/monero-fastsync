package server

import (
	"bytes"
	"net/http"
	"time"

	"github.com/exantech/moneroproto"

	"github.com/exantech/monero-fastsync/internal/app/fsd/rpc"
	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/metrics"
)

const (
	getBlocksUri = "/fastsync.bin"
	versionsUri  = "/fastsync_versions.bin"
)

var (
	currentlySupportedVersions = []uint32{1}
)

type Server struct {
	handler *BlocksHandler
}

func NewServer(handler *BlocksHandler) *Server {
	return &Server{
		handler: handler,
	}
}

func (s *Server) StartAsync(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc(getBlocksUri, WrapHandler(s.HandleGetBlocks))
	mux.HandleFunc(versionsUri, WrapHandler(s.HandleVersions))

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logging.Log.Fatalf("Failed to listen on address '%s': %s", address, err.Error())
		}
	}()
}

func WrapHandler(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		metrics.Rps.Mark(1)
		start := time.Now()

		logging.Log.Debugf("Incoming request %s", req.URL.Path)
		handler(resp, req)

		metrics.RequestDuration.UpdateSince(start)
	}
}

func (s *Server) HandleGetBlocks(resp http.ResponseWriter, req *http.Request) {
	ureq := rpc.GetMyBlocksRequest{}
	err := moneroproto.Read(req.Body, &ureq)
	if err != nil {
		logging.Log.Errorf("Failed to parse get blocks request: %s", err.Error())
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if ureq.Version != currentlySupportedVersions[0] {
		logging.Log.Errorf("Unsupported version %d", ureq.Version)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	ures := rpc.GetMyBlocksResponse{}
	res, err := s.handler.HandleGetBlocks(&ureq.Params)
	if err != nil {
		logging.Log.Errorf("Failed to process %s request: %s", getBlocksUri, err.Error())
		ures.Status = []byte(err.Error())

		writer := bytes.Buffer{}
		if e := moneroproto.Write(&writer, ures); e != nil {
			logging.Log.Errorf("Failed to serialize response: %s", e.Error())
			resp.WriteHeader(http.StatusInternalServerError)
		}

		if err == ErrRequestError {
			resp.WriteHeader(http.StatusBadRequest)
		} else {
			resp.WriteHeader(http.StatusInternalServerError)
		}

		resp.Write(writer.Bytes())
		return
	}

	ures.Status = []byte("ok")
	ures.Result = *res

	writer := bytes.Buffer{}
	if err := moneroproto.Write(&writer, ures); err != nil {
		logging.Log.Errorf("Failed to serialize response: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(writer.Bytes())
}

func (s *Server) HandleVersions(resp http.ResponseWriter, req *http.Request) {
	r := rpc.SupportedVersionsResponse{}
	r.Versions = currentlySupportedVersions

	moneroproto.Write(resp, r)
}
