package wart

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
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

func Create(configFile string, redisAddr string, redisPassword string, cluster string, wartName string, scriptList string, cpuThreshold float64, memThreshold float64, healthInterval time.Duration, host bool) (*Wart, error) {
	if configFile != "" {
		fBytes, err := ioutil.ReadFile(configFile)
		if err == nil {
			var f interface{}
			err2 := json.Unmarshal(fBytes, &f)
			if err2 == nil {
				m := f.(map[string]interface{})
				redisAddr = m["redis-address"].(string)
				redisPassword = m["redis-password"].(string)
				cluster = m["cluster"].(string)
				wartName = m["name"].(string)
				cpuThreshold = m["cpu-threshold"].(float64)
				memThreshold = m["mem-threshold"].(float64)
				t := m["health-interval"].(float64)
				healthInterval = time.Duration(t)
				host = m["host"].(bool)
			}
		}
	}

	w := &Wart{RedisAddr: redisAddr, RedisPassword: redisPassword,
		Cluster: cluster, WartName: wartName, ScriptList: scriptList,
		CpuThreshold: cpuThreshold, MemThreshold: memThreshold, HealthInterval: healthInterval, Healthy: true, SecondsTillDead: 1}

	if w.RedisAddr == "" {
		return nil, errors.New("no redis address provided")
	}

	w.Client = redis.NewClient(&redis.Options{
		Addr:     w.RedisAddr,
		Password: w.RedisPassword, // no password set
		DB:       0,               // use default DB
	})

	pong, pongErr := w.Client.Ping().Result()

	if pongErr != nil && pong != "PONG" {
		return nil, errors.New("redis failed ping")
	}

	w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "State", "online")
	w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "Status", "enabled")

	if w.ScriptList != "" {
		err := loadScripts(w, w.ScriptList)
		if err != nil {
			return nil, err
		}
	}

	if host {
		http.HandleFunc("/", w.handleEndpoint)
		go func() { http.ListenAndServe(":9999", nil) }()
	}
	return w, nil
}

//CheckHealth Checks the health of the wart and puts it in redis.
func CheckHealth(w *Wart) {
	if getCPUHealth(w) || getMemoryHealth(w) {
		w.Healthy = false
		w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "State", "critical")
		log.Error("Unhealthy")
	} else {
		if w.Healthy == false {
			log.Info("Health Restored")
		}
		w.Healthy = true
		w.Client.HSet(w.Cluster+":Warts:"+w.WartName, "State", "normal")
	}
}

func IsEnabled(w *Wart) bool {
	status := w.Client.HGet(w.Cluster+":Warts:"+w.WartName, "Status").Val()
	if status == "disabled" {
		return false
	}
	return true
}

func CheckThreads(w *Wart) {
	threads := w.Client.Keys(w.Cluster + ":Threads:*").Val()
	for i := range threads {
		threadStatus, threadState := getThreadStatus(w, threads[i])
		if threadStatus != "disabled" {
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
					takeThread(w, threads[i])
				}
			}
		}
	}
}

func (wart *Wart) handleEndpoint(w http.ResponseWriter, r *http.Request) {
	if wart.Healthy {
		key := wart.Cluster + ":Endpoints:" + html.EscapeString(r.URL.Path)
		source := wart.Client.HGet(key, "Source").Val()
		if source != "" {
			vm := otto.New()

			vm.Set("request", map[string]interface{}{
				"Method": r.Method,
				"Path":   html.EscapeString(r.URL.Path),
				"Query":  r.URL.Query(),
			})

			vm.Set("redis", map[string]interface{}{
				"Do": wart.Client.Do,
			})

			vm.Set("http", map[string]interface{}{
				"Get":      httpGet,
				"Post":     httpPost,
				"PostForm": httpPostForm,
				"Put":      httpPut,
				"Delete":   httpDelete,
			})

			vm.Set("response", map[string]interface{}{
				"Write": func(value string) {
					fmt.Fprintf(w, value)
				},
			})
			//Get whole script in memory.
			_, err := vm.Run(source)
			if err != nil {
				wart.Client.HSet(key, "Error", err.Error())
				wart.Client.HSet(key, "ErrorTime", time.Now())
				log.WithError(err).Error("Syntax error in script.")
			}
		} else {
			fmt.Fprintf(w, "No Endpoint")
		}
	}
}

func getThreadStatus(w *Wart, key string) (status string, state string) {
	status = w.Client.HGet(key, "Status").Val()
	state = w.Client.HGet(key, "State").Val()
	return
}

func loadScripts(w *Wart, scripts string) error {
	scriptArray := strings.Split(scripts, ",")
	for i := range scriptArray {
		scriptName := scriptArray[i]
		log.Info("Loading", scriptName)
		fBytes, err := ioutil.ReadFile(scriptName)
		if err != nil {
			return err
		}
		key := w.Cluster + ":Threads:" + scriptName
		w.Client.HSet(key, "Source", string(fBytes))
		w.Client.HSet(key, "Status", "enabled")
		w.Client.HSet(key, "State", "stopped")
		w.Client.HSet(key, "Heartbeat", 0)
		w.Client.HSet(key, "Owner", "")
		w.Client.HSet(key, "Error", "")
		w.Client.HSet(key, "ErrorTime", "")
	}

	return nil
}
func getMemoryHealth(w *Wart) bool {
	v, _ := mem.VirtualMemory()
	w.Client.HSet(w.Cluster+":Warts:"+w.WartName+":Health", "memory", v.UsedPercent)
	if v.UsedPercent > w.MemThreshold {
		return true
	}
	return false
}

func getCPUHealth(w *Wart) bool {
	c, _ := load.Avg()
	w.Client.HSet(w.Cluster+":Warts:"+w.WartName+":Health", "cpu", c.Load1)
	if c.Load1 > w.CpuThreshold {
		return true
	}
	return false
}

func takeThread(w *Wart, key string) {
	log.Info("Taking thread", key)
	source := w.Client.HGet(key, "Source").Val()
	w.Client.HSet(key, "State", "running")
	w.Client.HSet(key, "Heartbeat", time.Now().UnixNano())
	w.Client.HSet(key, "Owner", w.WartName)
	go thread(w, key, source)
}
func thread(w *Wart, key string, source string) {
	log.Info("Taking thread", key)
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
		w.Client.HSet(key, "Error", err.Error())
		w.Client.HSet(key, "ErrorTime", time.Now())
		log.WithError(err).Error("Syntax error in script.")
		return
	}

	//Run init script
	_, err = vm.Run("if (init != undefined) {init()}")
	if err != nil {
		w.Client.HSet(key, "State", "crashed")
		w.Client.HSet(key, "Status", "disabled")
		w.Client.HSet(key, "Error", err.Error())
		w.Client.HSet(key, "ErrorTime", time.Now())
		log.WithError(err).Error("Error running init() in script")
		return
	}
	for w.Healthy && !shouldStop {
		w.Client.HSet(key, "Heartbeat", time.Now().UnixNano())

		//Get status and stop if disabled.
		status := w.Client.HGet(key, "Status")
		owner := w.Client.HGet(key, "Owner")
		if status.Val() == "disabled" {
			log.Warn(key, "Was disabled.  Stopping thread.")
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
			w.Client.HSet(key, "Error", err.Error())
			w.Client.HSet(key, "ErrorTime", time.Now())
			log.WithError(err).Error("Error running main() in script")
			return
		}

		time.Sleep(time.Second * 1)
	}
	w.Client.HSet(key, "State", "stopped")
}
