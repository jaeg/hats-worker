package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var cluster = flag.String("cluster-name", "default", "name of cluster")
var wartName = flag.String("wart-name", "noname", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var cpuThreshold = flag.Float64("cpu-threshold", 1, "the load before unhealthy")
var memThreshold = flag.Float64("mem-threshold", 90.0, "max memory usage percent before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")
var runNow = flag.Bool("run-now", false, "Run loaded scripts immediately.")

var secondsTillDead = 1
var client *redis.Client
var clusterStatuses []string

var healthy = true

func main() {
	fmt.Println("Wart started.")
	err := start()
	if client != nil {
		defer client.HSet("Wart:"+*wartName, "Status", "offline")
	}

	defer fmt.Println("Wart stopped.")

	if err == nil {
		pong, pongErr := client.Ping().Result()

		if pongErr == nil && pong == "PONG" {
			go checkHealth()
			for true {
				checkThreads()
				client.HSet("Wart:"+*wartName, "Heartbeat", time.Now().UnixNano())
				time.Sleep(time.Second)
			}
		}
	} else {
		fmt.Println(err)
	}

}

func start() error {
	fmt.Println("Init")
	flag.Parse()

	if *redisAddr == "" {
		return errors.New("no redis address provided")
	}

	client = redis.NewClient(&redis.Options{
		Addr:     *redisAddr,
		Password: *redisPassword, // no password set
		DB:       0,              // use default DB
	})
	_, err := client.Ping().Result()
	if err != nil {
		return errors.New("redis failed ping")
	}
	client.HSet("Wart:"+*wartName, "Status", "online")

	if *scriptList != "" {
		scriptArray := strings.Split(*scriptList, ",")
		for i := range scriptArray {
			scriptName := scriptArray[i]
			fmt.Println("Loading " + scriptName)
			fBytes, err := ioutil.ReadFile(scriptName)
			if err != nil {
				return err
			}
			client.HSet(*cluster+":Threads:"+scriptName, "Source", string(fBytes))
			client.HSet(*cluster+":Threads:"+scriptName, "Status", "enabled")

			if *runNow {
				client.HSet(*cluster+":Threads:"+scriptName, "State", "running")
				client.HSet(*cluster+":Threads:"+scriptName, "Heartbeat", time.Now().UnixNano())
				client.HSet(*cluster+":Threads:"+scriptName, "Owner", *wartName)
				go thread(*cluster+":Threads:"+scriptName, string(fBytes))
			} else {
				client.HSet(*cluster+":Threads:"+scriptName, "State", "stopped")
				client.HSet(*cluster+":Threads:"+scriptName, "Heartbeat", 0)
				client.HSet(*cluster+":Threads:"+scriptName, "Owner", "")
			}

		}
	}

	return nil
}

func checkHealth() {
	//check to see if wart is healthy
	//If not figure out what it should give up
	//For each thing that should be given up compress code and put in redis
	for true {
		crit := false

		//CPU Load
		c, _ := load.Avg()
		fmt.Println("Current Load:", c.Load1)
		client.HSet("Wart:"+*wartName+":Health", "cpu", c.Load1)
		if c.Load1 > *cpuThreshold {
			crit = true
			fmt.Printf("Load Critical: %v\n", c.Load1)
		}

		//Memory Load
		v, _ := mem.VirtualMemory()
		fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)
		client.HSet("Wart:"+*wartName+":Health", "memory", v.UsedPercent)
		if v.UsedPercent > *memThreshold {
			crit = true
			fmt.Printf("Memory Critical - Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)
		}

		//Handle critcal condition
		if crit {
			healthy = false
			client.HSet("Wart:"+*wartName, "Status", "critical")
			fmt.Println("I'm unhealthy!")
		} else {
			healthy = true
			client.HSet("Wart:"+*wartName, "Status", "normal")
		}

		time.Sleep(*healthInterval * time.Second)
	}
}

func takeThread(key string) {
	fmt.Println("Taking thread: " + key)
	source := client.HGet(key, "Source").Val()
	client.HSet(key, "State", "running")
	client.HSet(key, "Heartbeat", time.Now().UnixNano())
	client.HSet(key, "Owner", *wartName)
	go thread(key, source)
}
func thread(key string, source string) {
	fmt.Println("Thread started: " + key)
	shouldStop := false
	vm := otto.New()

	vm.Set("redis", map[string]interface{}{
		"Do": client.Do,
	})

	vm.Set("http", map[string]interface{}{
		"Get":      httpGet,
		"Post":     httpPost,
		"PostForm": httpPostForm,
		"Put":      httpPut,
		"Delete":   httpDelete,
	})
	//Get whole script in memory.
	_, err := vm.Run(source)
	if err != nil {
		client.HSet(key, "State", "crashed")
		client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}

	//Run init script
	_, err = vm.Run("if (init != undefined) {init()}")
	if err != nil {
		client.HSet(key, "State", "crashed")
		client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}
	for healthy && !shouldStop {
		client.HSet(key, "Heartbeat", time.Now().UnixNano())

		//Get status and stop if disabled.
		status := client.HGet(key, "Status")
		owner := client.HGet(key, "Owner")
		if status.Val() == "disabled" {
			fmt.Println("Disabled:" + key)
			client.HSet(key, "State", "stopped")
			shouldStop = true
			continue
		}

		if owner.Val() != *wartName {
			shouldStop = true
			continue
		}

		_, err := vm.Run("if (main != undefined) {main()}")

		if err != nil {
			client.HSet(key, "State", "crashed")
			client.HSet(key, "Status", "disabled")
			fmt.Println(err)
			return
		}

		time.Sleep(time.Second * 1)
	}
	client.HSet(key, "State", "stopped")
}

func checkThreads() {
	if healthy {
		threads := client.Keys(*cluster + ":Threads:*").Val()
		for i := range threads {
			threadStatus := client.HGet(threads[i], "Status").Val()
			if threadStatus != "disabled" {
				threadState := client.HGet(threads[i], "State").Val()
				if threadState == "stopped" {
					takeThread(threads[i])
					continue
				}
				//Check to see if thread is hung or fell over before its state was updated
				lastHeartbeatString := client.HGet(threads[i], "Heartbeat").Val()
				lastHeartbeat, err := strconv.Atoi(lastHeartbeatString)
				if err == nil {
					elapsed := time.Since(time.Unix(0, int64(lastHeartbeat)))
					if int(elapsed.Seconds()) > secondsTillDead {
						fmt.Println("Dead thread: " + threads[i])
						//Take it.
						takeThread(threads[i])
					}
				}
			}
		}
	}

}
