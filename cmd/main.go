package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lvl484/positioning-filter/config"
	"github.com/lvl484/positioning-filter/storage"
)

const (
	shutdownTimeout = 10 * time.Second
)

var components []io.Closer

func main() {

	done := make(chan bool)

	configPath := flag.String("cp", "../config", "Path to config file")
	configName := flag.String("cn", "viper.config", "Name of config file")

	flag.Parse()

	viper, err := config.NewConfig(*configName, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	consulConfig := viper.NewConsulConfig()
	agentConfig := consulConfig.AgentConfig()
	consulClient, err := consulConfig.NewClient()

	if err != nil {
		log.Println(err)
		return
	}

	if err = consulConfig.ServiceRegister(consulClient, agentConfig); err != nil {
		log.Println(err)
		return
	}

	defer consulClient.Agent().ServiceDeregister(consulConfig.ServiceName)

	postgresConfig := viper.NewDBConfig()
	db, err := storage.Connect(postgresConfig)

	if err != nil {
		log.Println(err)
		return
	}

	components = append(components,
		//Put connection variables here
		db)

	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	log.Println("Recieved", sig, "signal")

	if err := gracefulShutdown(shutdownTimeout, done, components); err != nil {
		log.Println(err)
	}

	<-done

	log.Println("Service successfuly shutdown")
}
