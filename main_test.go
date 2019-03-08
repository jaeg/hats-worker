package main

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
)

func TestStartErrorWithNoRedisAddress(t *testing.T) {
	err := start(&wart{})
	if err.Error() != "no redis address provided" {
		t.Errorf("Did not fail due to no redis address.")
	}
}

func TestStartErrorWithFailedPing(t *testing.T) {
	err := start(&wart{redisAddr: "bad"})
	fmt.Println(err)
	if err.Error() != "redis failed ping" {
		t.Errorf("Did not fail due to failed ping.")
	}
}

func TestStartReturnsNilWhenSuccessful(t *testing.T) {
	mr, _ := miniredis.Run()
	err := start(&wart{redisAddr: mr.Addr()})
	if err != nil {
		t.Errorf("Errored starting wart.")
	}
}

func TestStartHandlesScriptsPassedIn(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "examples/hello.txt"
	err := start(&wart{redisAddr: mr.Addr(), scriptList: scripts})
	if err != nil {
		t.Errorf("Errored getting scripts")
	}
}

func TestStartErrorsIfItCanNotFindScript(t *testing.T) {
	mr, _ := miniredis.Run()
	scripts := "examples/doesnotexist.txt"
	err := start(&wart{redisAddr: mr.Addr(), scriptList: scripts})
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

	w := &wart{redisAddr: mr.Addr(), client: client}

	err := loadScripts(w, "examples/hello.txt")
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
	w := &wart{redisAddr: mr.Addr(), client: client}

	err := loadScripts(w, "examples/doesnotexist.txt")
	if err == nil {
		t.Errorf("Did not return error when script failed to load.")
	}
}
