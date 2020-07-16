package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaeg/hats-worker/worker"
	_ "github.com/robertkrimen/otto/underscore"
	log "github.com/sirupsen/logrus"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var cluster = flag.String("cluster-name", "default", "name of cluster")
var WorkerName = flag.String("worker-name", "", "the unique name of this worker")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var cpuThreshold = flag.Float64("cpu-threshold", 1, "the load before unhealthy")
var memThreshold = flag.Float64("mem-threshold", 90.0, "max memory usage percent before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")
var host = flag.Bool("host", false, "Allow this worker to be an http host.")
var hostPort = flag.String("host-port", "9999", "HTTP port of worker.")
var healthPort = flag.String("health-port", "8787", "Port to run health metrics on")
var configFile = flag.String("config", "", "Config file with worker settings")

func main() {
	rand.Seed(time.Now().UnixNano())
	var ctx = context.Background()
	log.SetLevel(log.InfoLevel)

	flag.Parse()
	w, err := worker.Create(*configFile, *redisAddr, *redisPassword, *cluster, *WorkerName, *scriptList, *host, *hostPort, *healthPort)

	//Capture sigterm
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		w.Shutdown()
	}()

	if err == nil {
		log.Info("worker Name: ", w.WorkerName)
		log.Debug("worker Started")
		//handle creating new threads.
		for worker.IsEnabled(w) {
			if w.Healthy {
				worker.CheckThreads(w)
				worker.CheckJobs(w)
			}
			w.Client.HSet(ctx, w.Cluster+":workers:"+w.WorkerName, "Heartbeat", time.Now().UnixNano())
			time.Sleep(time.Second)
		}
		log.Info("Shutting down.")
		if w.Client != nil {
			defer w.Client.HSet(ctx, w.Cluster+":workers:"+w.WorkerName, "State", "offline")
			defer log.Debug("worker Stopped")
		}
	} else {
		log.WithError(err).Error("Failed to start worker.")
	}
}
