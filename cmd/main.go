package main

import (
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lvl484/positioning-filter/logger"

	"github.com/lvl484/positioning-filter/config"
	"github.com/lvl484/positioning-filter/storage"
	log "github.com/sirupsen/logrus"
)

const (
	shutdownTimeout = 10 * time.Second
)

var components []io.Closer

func main() {

	configPath := flag.String("cp", "../config", "Path to config file")
	configName := flag.String("cn", "viper.config", "Name of config file")

	flag.Parse()

	viper, err := config.NewConfig(*configName, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	loggerConfig := viper.NewLoggerConfig()
	if err := logger.NewLogger(loggerConfig); err != nil {
		log.Println(err)
		return
	}

	consulConfig := viper.NewConsulConfig()
	agentConfig := consulConfig.AgentConfig()
	consulClient, err := consulConfig.NewClient()

	if err != nil {
		log.Error(err)
		return
	}

	if err = consulConfig.ServiceRegister(consulClient, agentConfig); err != nil {
		log.Error(err)
		return
	}

	defer consulClient.Agent().ServiceDeregister(consulConfig.ServiceName)

	postgresConfig := viper.NewDBConfig()
	db, err := storage.Connect(postgresConfig)

	if err != nil {
		log.Error(err)
		return
	}

	components = append(components,
		//Put connection variables here
		db)

	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	log.Info("Recieved", sig, "signal")

	if err := gracefulShutdown(shutdownTimeout, components); err != nil {
		log.Error(err)
	}

	log.Info("Service successfuly shutdown")
}
