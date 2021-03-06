package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/exantech/monero-fastsync/internal/app/fsd"
	"github.com/exantech/monero-fastsync/internal/app/fsd/server"
	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/metrics"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
	"github.com/exantech/monero-fastsync/pkg/genesis"
)

var (
	moduleName      = "fast-sync"
	version         = "develop"
	configPath      = flag.String("config", "fsd.yml", "path to configuration file")
	metricsConfPath = flag.String("metrics", "", "path to metrics config")
	help            = flag.Bool("h", false, "show this help message")
	ver             = flag.Bool("v", false, "show version")
)

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	if *ver {
		fmt.Println(version)
		return
	}

	if len(*configPath) == 0 {
		flag.Usage()
		log.Fatalf("Config path is required")
	}

	conf := fsd.MakeDefaultConfig()
	if err := utils.ReadYamlConfig(*configPath, &conf); err != nil {
		log.Fatalf("Couldn't read config file: %s", err.Error())
	}

	if err := logging.InitLogger(moduleName, conf.LogLevel); err != nil {
		log.Fatalf("Couldn't parse config: %s", err.Error())
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	if len(*metricsConfPath) != 0 {
		metricsConf := &fsd.MetricsConfig{}
		if err := utils.ReadJsonConfig(*metricsConfPath, metricsConf); err != nil {
			logging.Log.Fatalf("Failed to read metrics config %s: %s", *metricsConfPath, err.Error())
		}

		if err := metrics.Init(metricsConf.Graphite, metricsConf.Hostname); err != nil {
			logging.Log.Fatalf("Failed to init graphite: %s", err.Error())
		}

		logging.Log.Infof("Started with metrics config: host=%s, port=%d",
			metricsConf.Graphite.Host, metricsConf.Graphite.Port)
	}

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
		logging.Log.Fatal(http.ListenAndServe(conf.Pprof, nil))
	}()

	queue := server.NewJobsQueue(server.NewScanner(db), db, conf.ProcessBlocks, conf.ResultBlocks, conf.JobLifetime)

	err = queue.StartWorkers(conf.Workers)
	if err != nil {
		logging.Log.Fatalf("Failed to start async queue: %s", err.Error())
	}

	handler := server.NewServer(server.NewBlocksHandler(db, queue))

	logging.Log.Infof("Starting server on %s", conf.Server)
	handler.StartAsync(conf.Server)

	<-sig

	queue.Stop()
	logging.Log.Infof("Server stopped by signal")
}
