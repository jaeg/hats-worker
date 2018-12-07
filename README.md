# redis-wart
**W** - Widely
**A** - Accessible
**R** - Redis
**T** - Threading
**S** - System  
**A** simple interpreter designed to process data sitting in redis.

Each wart in a cluster checks in redis for work to do.  If it finds a stopped thread or a dead thread it takes the
thread and runs it locally.


## Runtime params
*cluster-name - name of cluster
*wart-name - name of the wart
*redis-address - address to redis server
*redis-password - password for redis server
-cpu-threshold - cpu usage over this means the wart is unhealthy
-mem-threshold - memory usage over this mean the wart is unhealthy
-scripts - scripts to register
-run-now - run registered scripts on this wart immediately

## Setup Instructions:
  Clone
  Run `go get`
  Run `go run *.go --redis-address=<address> --redis-password=<password --scripts=examples/main.txt,examples/hello.txt --wart-name=wart1`
