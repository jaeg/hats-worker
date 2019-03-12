package wart

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
)

func TestStartErrorWithNoRedisAddress(t *testing.T) {
	_, err := Create("", "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err.Error() != "no redis address provided" {
		t.Errorf("Did not fail due to no redis address.")
	}
}

func TestStartErrorWithFailedPing(t *testing.T) {
	_, err := Create("bad", "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err.Error() != "redis failed ping" {
		t.Errorf("Did not fail due to failed ping.")
	}
}

func TestStartReturnsNilWhenSuccessful(t *testing.T) {
	mr, _ := miniredis.Run()
	_, err := Create(mr.Addr(), "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err != nil {
		t.Errorf("Errored starting wart.")
	}
}

func TestStartHandlesScriptsPassedIn(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "../examples/hello.txt"
	_, err := Create(mr.Addr(), "", "TestCluster", "TestWart", scripts, 3, 3, 5)
	if err != nil {
		t.Errorf("Errored getting scripts")
	}
}

func TestStartErrorsIfItCanNotFindScript(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "../examples/doesnotexist.txt"
	_, err := Create(mr.Addr(), "", "TestCluster", "TestWart", scripts, 3, 3, 5)
	if err == nil {
		t.Errorf("Did not error getting scripts.")
	}
}

func TestLoadScripts(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})

	w := &Wart{RedisAddr: mr.Addr(), Client: client}

	err := loadScripts(w, "../examples/hello.txt")
	if err != nil {
		t.Errorf("Failed to load script.")
	}
}

func TestLoadScriptsDoesNotExist(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client}

	err := loadScripts(w, "../examples/doesnotexist.txt")
	if err == nil {
		t.Errorf("Did not return error when script failed to load.")
	}
}

func TestGetMemoryHealthBelowThreshold(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, MemThreshold: 150}

	value := getMemoryHealth(w)
	if value == true {
		t.Errorf("Memory false flagged")
	}
}

func TestGetMemoryHealthAboveThreshold(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, MemThreshold: 0}

	value := getMemoryHealth(w)
	if value == false {
		t.Errorf("Memory did not flag")
	}
}

func TestGetCPUHealthBelowThreshold(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, CpuThreshold: 150}

	value := getCPUHealth(w)
	if value == true {
		t.Errorf("CPU false flagged")
	}
}

func TestGetCPUHealthAboveThreshold(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, CpuThreshold: 0}

	value := getCPUHealth(w)
	if value == false {
		t.Errorf("CPU did not flag")
	}
}

func TestCheckHealthIsHealthy(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, CpuThreshold: 100, MemThreshold: 100}
	CheckHealth(w)
	if w.Healthy == false {
		t.Errorf("False healthy flag")
	}
}

func TestCheckHealthIsUnHealthy(t *testing.T) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0, // use default DB
	})
	w := &Wart{RedisAddr: mr.Addr(), Client: client, CpuThreshold: 0, MemThreshold: 0}
	CheckHealth(w)
	if w.Healthy == true {
		t.Errorf("Failed to flag unhealthy")
	}
}
