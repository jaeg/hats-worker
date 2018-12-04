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
	"github.com/shirou/gopsutil/load"
)

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var cluster = flag.String("cluster", "default", "name of cluster")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var wartName = flag.String("wart-name", "noname", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var criticalLoad = flag.Float64("max-cpu", 1, "the load before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")
var runNow = flag.Bool("run-now", false, "Run loaded scripts immediately.")
var secondsTillDead = 1
var client *redis.Client
var clusterStatuses []string

var isCrit = false

func main() {
	fmt.Println("Wart started.")
	err := Init()
	if client != nil {
		defer client.Set("Status:"+*wartName, "offline", 100)
	}

	defer fmt.Println("Wart stopped.")

	if err == nil {
		pong, err := client.Ping().Result()
		fmt.Println(pong, err)

		if err == nil {
			//go checkHealth()
			for true {
				checkThreads()
				time.Sleep(time.Second)
			}
		}
	} else {
		fmt.Println(err)
	}

}

func Init() error {
	flag.Parse()

	if *redisAddr == "" {
		return errors.New("No redis address provided.")
	}

	client = redis.NewClient(&redis.Options{
		Addr:     *redisAddr,
		Password: *redisPassword, // no password set
		DB:       0,              // use default DB
	})
	client.Set(*cluster+":Warts:"+*wartName+":Status", "online", *healthInterval*time.Second)

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
				client.HSet(*cluster+":Threads:"+scriptName, "Owner", "")
				go thread(*cluster+":Threads:"+scriptName, string(fBytes))
			} else {
				fmt.Println("Initial:" + *cluster + ":Threads:" + scriptName)
				client.HSet(*cluster+":Threads:"+scriptName, "State", "stopped")
				client.HSet(*cluster+":Threads:"+scriptName, "Heartbeat", 0)
				client.HSet(*cluster+":Threads:"+scriptName, "Owner", *wartName)
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
		fmt.Println("Health Check")
		crit := false

		c, _ := load.Avg()
		fmt.Println("Current Load:", c.Load1)
		if c.Load1 > *criticalLoad {
			crit = true
			fmt.Printf("Load Critical: %v\n", c.Load1)
		}

		if crit {
			isCrit = true
			client.Set("Status:"+*wartName, "crit", *healthInterval*time.Second)
			fmt.Println("I'm unhealthy!")
		}

		if crit == false {
			client.Set("Status:"+*wartName, "online", *healthInterval*time.Second)
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
	for !isCrit {
		client.HSet(key, "Heartbeat", time.Now().UnixNano())

		//Get status and stop if disabled.
		status := client.HGet(key, "Status")
		owner := client.HGet(key, "Owner")
		if status.Val() == "disabled" {
			fmt.Println("Disabled:" + key)
			client.HSet(key, "State", "stopped")
			return
		}

		if owner.Val() != *wartName {
			return
		}

		//Run script here

		time.Sleep(time.Second * 1)
	}
}

func checkThreads() {
	threads := client.Keys(*cluster + ":Threads:*").Val()
	for i := range threads {
		fmt.Println(threads[i])
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
				fmt.Println(elapsed)
				if int(elapsed.Seconds()) > secondsTillDead {
					fmt.Println("Dead thread: " + threads[i])
					//Take it.
					takeThread(threads[i])
				}
			}
		}
	}
}
