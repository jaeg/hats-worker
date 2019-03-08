package wart

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
)

func TestStartErrorWithNoRedisAddress(t *testing.T) {
	_, err := CreateWart("", "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err.Error() != "no redis address provided" {
		t.Errorf("Did not fail due to no redis address.")
	}
}

func TestStartErrorWithFailedPing(t *testing.T) {
	_, err := CreateWart("bad", "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err.Error() != "redis failed ping" {
		t.Errorf("Did not fail due to failed ping.")
	}
}

func TestStartReturnsNilWhenSuccessful(t *testing.T) {
	mr, _ := miniredis.Run()
	_, err := CreateWart(mr.Addr(), "", "TestCluster", "TestWart", "", 3, 3, 5)
	if err != nil {
		t.Errorf("Errored starting wart.")
	}
}

func TestStartHandlesScriptsPassedIn(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "../examples/hello.txt"
	_, err := CreateWart(mr.Addr(), "", "TestCluster", "TestWart", scripts, 3, 3, 5)
	if err != nil {
		t.Errorf("Errored getting scripts")
	}
}

func TestStartErrorsIfItCanNotFindScript(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "../examples/doesnotexist.txt"
	_, err := CreateWart(mr.Addr(), "", "TestCluster", "TestWart", scripts, 3, 3, 5)
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
