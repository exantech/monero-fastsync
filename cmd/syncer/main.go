package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/exantech/monero-fastsync/internal/app/syncer"
	"github.com/exantech/monero-fastsync/internal/app/syncer/worker"
	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/monero-fastsync/pkg/genesis"
)

var (
	moduleName = "syncer"
	configPath = flag.String("config", "syncer.yml", "path to configuration file")
	initDb     = flag.Bool("init", false, "initially populate DB with genesis block info")
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

	var conf syncer.Config
	if err := utils.ReadConfig(*configPath, &conf); err != nil {
		log.Fatalf("Couldn't read config file: %s", err.Error())
	}

	if err := logging.InitLogger(moduleName, conf.LogLevel); err != nil {
		log.Fatalf("Couln't parse config: %s", err.Error())
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Fatal(http.ListenAndServe(conf.Pprof, nil))
	}()

	ctx, cancel := context.WithCancel(context.Background())

	logging.Log.Info("Creating new db connection")
	db, err := worker.NewDbOperator(conf.BlockchainDb)
	if err != nil {
		logging.Log.Fatalf("Couldn't make db fetcher: %s", err.Error())
	}

	node, err := worker.NewNodeFetcher(conf.NodeAddress)
	if err != nil {
		logging.Log.Fatalf("Couldn't make node fetcher: %s", err.Error())
	}

	logging.Log.Infof("Using %s network", strings.ToUpper(conf.Network))

	genesisInfo := genesis.GetGenesisBlockInfo(conf.Network)
	w := worker.NewWorker(db, node, genesisInfo)

	logging.Log.Info("Checking genesis block hash")
	if err = w.CheckGenesis(ctx, *initDb); err != nil {
		log.Fatalf("Error on checking genesis block hash compliance: %s. Make sure you are either connected to DB"+
			" with the same blockchain or set up DB settings properly. If you run syncer with empty DB for the first time"+
			" launch it with the '-init' flag to save genesis block into database", err.Error())
	}

	if *initDb {
		logging.Log.Infof("Genesis successfully saved. Now you may start syncer without '-init' option")
		return
	}

	logging.Log.Info("Starting sync loop")
	done := w.RunSyncLoop(ctx)

loop:
	for {
		select {
		case <-sig:
			logging.Log.Info("Sending stop signal to the worker")
			cancel()
		case err := <-done:
			if err == worker.ErrInterrupted {
				logging.Log.Info("Sync loop interrupted")
			} else if err != nil {
				logging.Log.Fatalf("Sync worker finished with error: %s", err.Error())
			} else {
				logging.Log.Info("Sync worker stopped")
			}

			break loop
		}
	}
}
