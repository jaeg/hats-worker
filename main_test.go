package main

import (
	"testing"

	"github.com/alicebob/miniredis"
)

func TestStartErrorWithNoRedisAddress(t *testing.T) {
	err := start()
	if err.Error() != "no redis address provided" {
		t.Errorf("Did not fail due to no redis address.")
	}
}

func TestStartErrorWithFailedPing(t *testing.T) {
	addr := "bad"
	redisAddr = &addr
	err := start()
	if err.Error() != "redis failed ping" {
		t.Errorf("Did not fail due to failed ping.")
	}
}

func TestStartReturnsNilWhenSuccessful(t *testing.T) {
	mr, _ := miniredis.Run()
	addr := mr.Addr()
	redisAddr = &addr
	err := start()
	if err != nil {
		t.Errorf("Errored starting wart.")
	}
}

func TestStartHandlesScriptsPassedIn(t *testing.T) {
	mr, _ := miniredis.Run()
	addr := mr.Addr()
	redisAddr = &addr

	scripts := "examples/hello.txt"
	scriptList = &scripts
	err := start()
	if err != nil {
		t.Errorf("Errored getting scripts")
	}
}

func TestStartHandlesScriptsPassedInAndCanRunNow(t *testing.T) {
	mr, _ := miniredis.Run()
	addr := mr.Addr()
	redisAddr = &addr

	scripts := "examples/hello.txt"
	scriptList = &scripts

	run := true
	runNow = &run
	err := start()
	if err != nil {
		t.Errorf("Errored getting scripts")
	}
}

func TestStartErrorsIfItCanNotFindScript(t *testing.T) {
	mr, _ := miniredis.Run()
	addr := mr.Addr()
	redisAddr = &addr

	scripts := "examples/doesnotexist.txt"
	scriptList = &scripts
	err := start()
	if err == nil {
		t.Errorf("Did not error getting scripts.")
	}
}
