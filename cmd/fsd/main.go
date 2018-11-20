package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/exantech/monero-fastsync/internal/app/fsd"
	"github.com/exantech/monero-fastsync/internal/app/fsd/server"
	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/monero-fastsync/pkg/genesis"
)

var (
	moduleName = "fast-sync"
	configPath = flag.String("config", "fsd.yml", "path to configuration file")
	help       = flag.Bool("h", false, "show this help message")
)

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	if len(*configPath) == 0 {
		flag.Usage()
		log.Fatalf("Config path is required")
	}

	var conf fsd.Config
	if err := utils.ReadConfig(*configPath, &conf); err != nil {
		log.Fatalf("Couldn't read config file: %s", err.Error())
	}

	if err := logging.InitLogger(moduleName, conf.LogLevel); err != nil {
		log.Fatalf("Couldn't parse config: %s", err.Error())
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	logging.Log.Infof("Connecting to DB: host=%s, port=%d, database=%s, user=%s",
		conf.BlockchainDb.Host, conf.BlockchainDb.Port, conf.BlockchainDb.Database, conf.BlockchainDb.User)

	logging.Log.Infof("Using %s network", strings.ToUpper(conf.Network))

	db, err := server.NewDbWorker(conf.BlockchainDb)
	if err != nil {
		logging.Log.Fatalf("Failed to connect to DB: %s", err.Error())
	}

	currentGenesis, err := db.GetBlockEntry(0)
	if err != nil {
		logging.Log.Fatalf("Failed to get genesis block: %s", err.Error())
	}

	if currentGenesis.Hash != genesis.GetGenesisBlockInfo(conf.Network).Hash {
		logging.Log.Fatalf("Genesis mismatch. Current: %s, expected: %s",
			currentGenesis.Hash, genesis.GetGenesisBlockInfo(conf.Network).Hash)
	}

	logging.Log.Infof("Starting pprof on %s", conf.Pprof)
	go func() {
		log.Fatal(http.ListenAndServe(conf.Pprof, nil))
	}()

	handler := server.NewServer(server.NewBlocksHandler(db, server.NewScanner(db)))

	logging.Log.Infof("Starting server on %s", conf.Server)
	handler.StartAsync(conf.Server)

	<-sig

	logging.Log.Infof("Server stopped by signal")
}