package wart

import (
	"errors"
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

type Wart struct {
	RedisAddr       string
	RedisPassword   string
	Cluster         string
	WartName        string
	ScriptList      string
	CpuThreshold    float64
	MemThreshold    float64
	HealthInterval  time.Duration
	Client          *redis.Client
	Healthy         bool
	ThreadCount     int
	SecondsTillDead int
}

func Start(w *Wart) error {
	if w.RedisAddr == "" {
		return errors.New("no redis address provided")
	}

	w.Client = redis.NewClient(&redis.Options{
		Addr:     w.RedisAddr,
		Password: w.RedisPassword, // no password set
		DB:       0,               // use default DB
	})

	pong, pongErr := w.Client.Ping().Result()

	if pongErr != nil && pong != "PONG" {
		return errors.New("redis failed ping")
	}

	w.Client.HSet("Wart:"+w.WartName, "Status", "online")

	if w.ScriptList != "" {
		err := loadScripts(w, w.ScriptList)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadScripts(w *Wart, scripts string) error {
	scriptArray := strings.Split(scripts, ",")
	for i := range scriptArray {
		scriptName := scriptArray[i]
		fmt.Println("Loading " + scriptName)
		fBytes, err := ioutil.ReadFile(scriptName)
		if err != nil {
			return err
		}
		w.Client.HSet(w.Cluster+":Threads:"+scriptName, "Source", string(fBytes))
		w.Client.HSet(w.Cluster+":Threads:"+scriptName, "Status", "enabled")
		w.Client.HSet(w.Cluster+":Threads:"+scriptName, "State", "stopped")
		w.Client.HSet(w.Cluster+":Threads:"+scriptName, "Heartbeat", 0)
		w.Client.HSet(w.Cluster+":Threads:"+scriptName, "Owner", "")
	}

	return nil
}

func CheckHealth(w *Wart) {
	//check to see if wart is Healthy
	//If not figure out what it should give up
	//For each thing that should be given up compress code and put in redis
	//Handle critcal condition
	if getCPUHealth(w, w.CpuThreshold) || getMemoryHealth(w, w.MemThreshold) {
		w.Healthy = false
		w.Client.HSet("Wart:"+w.WartName, "Status", "critical")
		fmt.Println("I'm unHealthy!")
	} else {
		w.Healthy = true
		w.Client.HSet("Wart:"+w.WartName, "Status", "normal")
	}

	time.Sleep(w.HealthInterval * time.Second)

}

func getMemoryHealth(w *Wart, threshold float64) bool {
	v, _ := mem.VirtualMemory()
	w.Client.HSet("Wart:"+w.WartName+":Health", "memory", v.UsedPercent)
	if v.UsedPercent > threshold {
		return true
	}
	return false
}

func getCPUHealth(w *Wart, threshold float64) bool {
	c, _ := load.Avg()
	w.Client.HSet("Wart:"+w.WartName+":Health", "cpu", c.Load1)
	if c.Load1 > threshold {
		return true
	}
	return false
}

func takeThread(w *Wart, key string) {
	fmt.Println("Taking thread: " + key)
	source := w.Client.HGet(key, "Source").Val()
	w.Client.HSet(key, "State", "running")
	w.Client.HSet(key, "Heartbeat", time.Now().UnixNano())
	w.Client.HSet(key, "Owner", w.WartName)
	go thread(w, key, source)
}
func thread(w *Wart, key string, source string) {
	fmt.Println("Thread started: " + key)
	w.ThreadCount++
	shouldStop := false
	vm := otto.New()

	vm.Set("redis", map[string]interface{}{
		"Do": w.Client.Do,
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
		w.Client.HSet(key, "State", "crashed")
		w.Client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}

	//Run init script
	_, err = vm.Run("if (init != undefined) {init()}")
	if err != nil {
		w.Client.HSet(key, "State", "crashed")
		w.Client.HSet(key, "Status", "disabled")
		fmt.Println(err)
		return
	}
	for w.Healthy && !shouldStop {
		w.Client.HSet(key, "Heartbeat", time.Now().UnixNano())

		//Get status and stop if disabled.
		status := w.Client.HGet(key, "Status")
		owner := w.Client.HGet(key, "Owner")
		if status.Val() == "disabled" {
			fmt.Println("Disabled:" + key)
			w.Client.HSet(key, "State", "stopped")
			shouldStop = true
			continue
		}

		if owner.Val() != w.WartName {
			shouldStop = true
			continue
		}

		_, err := vm.Run("if (main != undefined) {main()}")

		if err != nil {
			w.Client.HSet(key, "State", "crashed")
			w.Client.HSet(key, "Status", "disabled")
			fmt.Println(err)
			return
		}

		time.Sleep(time.Second * 1)
	}
	w.Client.HSet(key, "State", "stopped")
	w.ThreadCount--
}

func CheckThreads(w *Wart) {
	if w.Healthy {
		threads := w.Client.Keys(w.Cluster + ":Threads:*").Val()
		for i := range threads {
			threadStatus := w.Client.HGet(threads[i], "Status").Val()
			if threadStatus != "disabled" {
				threadState := w.Client.HGet(threads[i], "State").Val()
				if threadState == "stopped" {
					takeThread(w, threads[i])
					continue
				}
				//Check to see if thread is hung or fell over before its state was updated
				lastHeartbeatString := w.Client.HGet(threads[i], "Heartbeat").Val()
				lastHeartbeat, err := strconv.Atoi(lastHeartbeatString)
				if err == nil {
					elapsed := time.Since(time.Unix(0, int64(lastHeartbeat)))
					if int(elapsed.Seconds()) > w.SecondsTillDead {
						fmt.Println("Dead thread: " + threads[i])
						//Take it.
						takeThread(w, threads[i])
					}
				}
			}
		}
	}

}
