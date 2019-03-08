package main

import (
	"flag"
	"fmt"
	"redis-wart/wart"
	"time"

	_ "github.com/robertkrimen/otto/underscore"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var cluster = flag.String("cluster-name", "default", "name of cluster")
var wartName = flag.String("wart-name", "noname", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var cpuThreshold = flag.Float64("cpu-threshold", 1, "the load before unhealthy")
var memThreshold = flag.Float64("mem-threshold", 90.0, "max memory usage percent before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")

func main() {
	flag.Parse()

	w, err := wart.CreateWart(*redisAddr, *redisPassword, *cluster, *wartName, *scriptList, *cpuThreshold, *memThreshold, *healthInterval)
	if w.Client != nil {
		defer w.Client.HSet("Wart:"+w.WartName, "Status", "offline")
	}

	if err == nil {
		//Health check thread
		go func() {
			for true {
				wart.CheckHealth(w)
			}
		}()

		//handle creating new threads.
		for true {
			wart.CheckThreads(w)
			w.Client.HSet("Wart:"+w.WartName, "Heartbeat", time.Now().UnixNano())
			time.Sleep(time.Second)
		}
	} else {
		fmt.Println("Failed to start:", err)
	}

}
