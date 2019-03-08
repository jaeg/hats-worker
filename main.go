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

type wart struct {
	redisAddr       string
	redisPassword   string
	cluster         string
	wartName        string
	scriptList      string
	cpuThreshold    float64
	memThreshold    float64
	healthInterval  time.Duration
	client          *redis.Client
	healthy         bool
	threadCount     int
	secondsTillDead int
}

func main() {
	flag.Parse()
	fmt.Println("Wart started.")
	w := &wart{redisAddr: *redisAddr, redisPassword: *redisPassword,
		cluster: *cluster, wartName: *wartName, scriptList: *scriptList,
		cpuThreshold: *cpuThreshold, memThreshold: *memThreshold, healthInterval: *healthInterval, healthy: true, secondsTillDead: 1}
	var err error
	fmt.Println(*redisAddr,w)
	err = start(w)
	if w.client != nil {
		defer w.client.HSet("Wart:"+w.wartName, "Status", "offline")
	}

	defer fmt.Println("Wart stopped.")

	if err == nil {
		go checkHealth(w)
		for true {
			checkThreads(w)
			w.client.HSet("Wart:"+w.wartName, "Heartbeat", time.Now().UnixNano())
			time.Sleep(time.Second)
		}
	} else {
		fmt.Println(err)
	}

}

func start(w *wart) error{
	fmt.Println("Init")


	if w.redisAddr == "" {
		return errors.New("no redis address provided")
	}

	w.client = redis.NewClient(&redis.Options{
		Addr:     w.redisAddr,
		Password: w.redisPassword, // no password set
		DB:       0,              // use default DB
	})

	pong, pongErr := w.client.Ping().Result()

	if pongErr != nil && pong != "PONG" {
		return errors.New("redis failed ping")
	}

	w.client.HSet("Wart:"+w.wartName, "Status", "online")

	if w.scriptList != "" {
		err := loadScripts(w, w.scriptList)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadScripts(w *wart, scripts string) error {
	scriptArray := strings.Split(scripts, ",")
	for i := range scriptArray {
		scriptName := scriptArray[i]
		fmt.Println("Loading " + scriptName)
		fBytes, err := ioutil.ReadFile(scriptName)
		if err != nil {
			return err
		}
		w.client.HSet(w.cluster+":Threads:"+scriptName, "Source", string(fBytes))
		w.client.HSet(w.cluster+":Threads:"+scriptName, "Status", "enabled")
		w.client.HSet(w.cluster+":Threads:"+scriptName, "State", "stopped")
		w.client.HSet(w.cluster+":Threads:"+scriptName, "Heartbeat", 0)
		w.client.HSet(w.cluster+":Threads:"+scriptName, "Owner", "")
	}

	return nil
}

func checkHealth(w *wart) {
	//check to see if wart is healthy
	//If not figure out what it should give up
	//For each thing that should be given up compress code and put in redis
	for true {
		//Handle critcal condition
		if getCPUHealth(w, w.cpuThreshold) || getMemoryHealth(w, w.memThreshold) {
			w.healthy = false
			w.client.HSet("Wart:"+w.wartName, "Status", "critical")
			fmt.Println("I'm unhealthy!")
		} else {
			w.healthy = true
			w.client.HSet("Wart:"+w.wartName, "Status", "normal")
		}

		time.Sleep(w.healthInterval * time.Second)
	}
}

func getMemoryHealth(w *wart, threshold float64) bool {
	v, _ := mem.VirtualMemory()
	w.client.HSet("Wart:"+w.wartName+":Health", "memory", v.UsedPercent)
	if v.UsedPercent > threshold {
		return true
	}
	return false
}

func getCPUHealth(w *wart, threshold float64) bool {
	c, _ := load.Avg()
	w.client.HSet("Wart:"+w.wartName+":Health", "cpu", c.Load1)
	if c.Load1 > threshold {
		return true
	}
	return false
}

func takeThread(w *wart, key string) {
	fmt.Println("Taking thread: " + key)
	source := w.client.HGet(key, "Source").Val()
	w.client.HSet(key, "State", "running")
	w.client.HSet(key, "Heartbeat", time.Now().UnixNano())
	w.client.HSet(key, "Owner", w.wartName)
	go thread(w, key, source)
}
func thread(w *wart, key string, source string) {
	fmt.Println("Thread started: " + key)
	w.threadCount++
	shouldStop := false
	vm := otto.New()

	vm.Set("redis", map[string]interface{}{
		"Do": w.client.Do,
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
		w.client.HSet(key, "State", "crashed")
		w.client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}

	//Run init script
	_, err = vm.Run("if (init != undefined) {init()}")
	if err != nil {
		w.client.HSet(key, "State", "crashed")
		w.client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}
	for w.healthy && !shouldStop {
		w.client.HSet(key, "Heartbeat", time.Now().UnixNano())

		//Get status and stop if disabled.
		status := w.client.HGet(key, "Status")
		owner := w.client.HGet(key, "Owner")
		if status.Val() == "disabled" {
			fmt.Println("Disabled:" + key)
			w.client.HSet(key, "State", "stopped")
			shouldStop = true
			continue
		}

		if owner.Val() != w.wartName {
			shouldStop = true
			continue
		}

		_, err := vm.Run("if (main != undefined) {main()}")

		if err != nil {
			w.client.HSet(key, "State", "crashed")
			w.client.HSet(key, "Status", "disabled")
			fmt.Println(err)
			return
		}

		time.Sleep(time.Second * 1)
	}
	w.client.HSet(key, "State", "stopped")
	w.threadCount--
}

func checkThreads(w *wart) {
	if w.healthy {
		threads := w.client.Keys(w.cluster + ":Threads:*").Val()
		for i := range threads {
			threadStatus := w.client.HGet(threads[i], "Status").Val()
			if threadStatus != "disabled" {
				threadState := w.client.HGet(threads[i], "State").Val()
				if threadState == "stopped" {
					takeThread(w, threads[i])
					continue
				}
				//Check to see if thread is hung or fell over before its state was updated
				lastHeartbeatString := w.client.HGet(threads[i], "Heartbeat").Val()
				lastHeartbeat, err := strconv.Atoi(lastHeartbeatString)
				if err == nil {
					elapsed := time.Since(time.Unix(0, int64(lastHeartbeat)))
					if int(elapsed.Seconds()) > w.secondsTillDead {
						fmt.Println("Dead thread: " + threads[i])
						//Take it.
						takeThread(w, threads[i])
					}
				}
			}
		}
	}

}
