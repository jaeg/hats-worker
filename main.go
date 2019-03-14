package main

import (
	"flag"
	"redis-wart/wart"
	"time"

	_ "github.com/robertkrimen/otto/underscore"
	log "github.com/sirupsen/logrus"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var cluster = flag.String("cluster-name", "default", "name of cluster")
var wartName = flag.String("wart-name", "noname", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var cpuThreshold = flag.Float64("cpu-threshold", 1, "the load before unhealthy")
var memThreshold = flag.Float64("mem-threshold", 90.0, "max memory usage percent before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")
var host = flag.Bool("host", false, "Allow this wart to be an http host.")
var configFile = flag.String("config", "", "Config file with wart settings")

func main() {
	log.SetLevel(log.InfoLevel)
	log.Debug("Wart Started")
	flag.Parse()
	w, err := wart.Create(*configFile, *redisAddr, *redisPassword, *cluster, *wartName, *scriptList, *cpuThreshold, *memThreshold, *healthInterval, *host)
	if w.Client != nil {
		defer w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "State", "offline")
		defer log.Debug("Wart Stopped")
	}

	if err == nil {
		//Health check thread
		go func() {
			for true {
				wart.CheckHealth(w)
				time.Sleep(w.HealthInterval * time.Second)
			}
		}()

		//handle creating new threads.
		for wart.IsEnabled(w) {
			if w.Healthy {
				wart.CheckThreads(w)
			}
			w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "Heartbeat", time.Now().UnixNano())
			time.Sleep(time.Second)
		}
		log.Info("Wart has been disabled. Shutting down.")
	} else {
		log.WithError(err).Error("Failed to start wart.")
	}

}
