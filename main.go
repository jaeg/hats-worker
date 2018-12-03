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

type Script struct {
	Source  string
	Running bool
}

var redisAddr = flag.String("redis-address", "", "the address for the main redis")
var cluster = flag.String("cluster", "default", "name of cluster")
var redisPassword = flag.String("redis-password", "", "the password for redis")
var wartName = flag.String("wart-name", "noname", "the unique name of this wart")
var scriptList = flag.String("scripts", "", "comma delimited list of scripts to run")
var criticalLoad = flag.Float64("max-cpu", 1, "the load before unhealthy")
var healthInterval = flag.Duration("health-interval", 5, "Seconds delay for health check")

var secondsTillDead = 1
var client *redis.Client
var clusterStatuses []string
var scripts map[string]*Script

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

	scripts = make(map[string]*Script)

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
			fBytes, err := ioutil.ReadFile(scriptName)
			if err != nil {
				return err
			}
			client.Set(*cluster+":Threads:"+scriptName+":Source", string(fBytes), 0)
			client.Set(*cluster+":Threads:"+scriptName+":State", "running", 0)
			client.Set(*cluster+":Threads:"+scriptName+":Status", "enabled", 0)
			client.Set(*cluster+":Threads:"+scriptName, time.Now().UnixNano(), 0)

			go job(scriptName, string(fBytes))
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

func job(name string, source string) {
	shouldStop := false
	for !shouldStop {
		client.Set(*cluster+":Threads:"+name, time.Now().UnixNano(), 0)
		time.Sleep(time.Second * 3)
	}
}

func whatToGiveUp() map[string]string {
	//Needs to figure out what script it can stop and give to another server.
	giveUpScripts := make(map[string]string)

	for k, v := range scripts {
		if v.Running == false {
			giveUpScripts[k] = v.Source
			delete(scripts, k)
		}
	}
	return giveUpScripts
}

func checkThreads() {
	threads := client.Keys(*cluster + ":Threads:*").Val()
	for i := range threads {
		lastHeartbeatString := client.Get(threads[i]).Val()
		lastHeartbeat, err := strconv.Atoi(lastHeartbeatString)
		if err == nil {
			fmt.Println("Working on: " + threads[i])
			elapsed := time.Since(time.Unix(0, int64(lastHeartbeat)))
			fmt.Println(elapsed)
			if int(elapsed.Seconds()) > secondsTillDead {
				fmt.Println("Dead thread: " + threads[i])
			}
		}
	}
}
