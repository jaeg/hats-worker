package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaeg/redis-wart/wart"

	_ "github.com/robertkrimen/otto/underscore"
	log "github.com/sirupsen/logrus"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var cluster = flag.String("cluster-name", "default", "name of cluster")
var wartName = flag.String("wart-name", "", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var cpuThreshold = flag.Float64("cpu-threshold", 1, "the load before unhealthy")
var memThreshold = flag.Float64("mem-threshold", 90.0, "max memory usage percent before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")
var host = flag.Bool("host", false, "Allow this wart to be an http host.")
var healthPort = flag.String("health-port", "8787", "Port to run health metrics on")
var configFile = flag.String("config", "", "Config file with wart settings")

func main() {
	var ctx = context.Background()
	log.SetLevel(log.InfoLevel)

	flag.Parse()
	w, err := wart.Create(*configFile, *redisAddr, *redisPassword, *cluster, *wartName, *scriptList, *host, *healthPort)

	//Capture sigterm
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		w.Shutdown()
	}()

	log.Info("Wart Name: ", w.WartName)
	log.Debug("Wart Started")
	if err == nil {
		//handle creating new threads.
		for wart.IsEnabled(w) {
			if w.Healthy {
				wart.CheckThreads(w)
			}
			w.Client.HSet(ctx, w.Cluster+":Warts:"+w.WartName, "Heartbeat", time.Now().UnixNano())
			time.Sleep(time.Second)
		}
		log.Info("Shutting down.")
	} else {
		log.WithError(err).Error("Failed to start wart.")
	}

	if w.Client != nil {
		defer w.Client.HSet(ctx, w.Cluster+":Warts:"+w.WartName, "State", "offline")
		defer log.Debug("Wart Stopped")
	}

}
