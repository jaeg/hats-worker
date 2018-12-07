# redis-wart
**W** - Widely  
**A** - Accessible  
**R** - Redis   
**T** - Threading    
**S** - System  
A simple interpreter designed to process data sitting in redis.

Each wart in a cluster checks in redis for work to do.  If it finds a stopped thread or a dead thread it takes the thread and runs it locally.

## Runtime params
- cluster-name - name of cluster   
- wart-name - name of the wart   
- redis-address - address to redis server  
- redis-password - password for redis server   
- cpu-threshold - cpu usage over this means the wart is unhealthy   
- mem-threshold - memory usage over this mean the wart is unhealthy  
- scripts - scripts to register  
- run-now - run registered scripts on this wart immediately

## Setup Instructions:
  - `git clone https://github.com/jaeg/redis-wart.git`
  - `go get`
  - `go run *.go --redis-address=<address> --redis-password=<password --scripts=examples/main.txt,examples/hello.txt --wart-name=wart1`

## Javascript implementation
Wart's Javascript implementation is based on [Otto](https://github.com/robertkrimen/otto).  Each thread maintains its own scope.  When a thread starts it runs the entire script.  It then runs `init()` if present in the source code.  If present a thread will call `main()` after confirming the thread is still running.

### Wart Todo
- [x] - Run a thread from redis.
- [x] - Create thread from file.
- [x] - Stop thread if wart is unhealthy.
- [x] - Stop thread if status is disabled.
- [x] - Stop thread if not the owner of thread.
- [x] - CPU health check based on threshold.
- [ ] - Memory health check based on threshold.
- [ ] - Recover from critical state when thresholds are met.

### Javascript Todo
- [x] - Basic javascript implementation
- [x] - Keep scope inside of thread
- [ ] - Redis wrapper
- [ ] - Wart information i.e. Health, name, cluster
- [ ] - Thread information i.e. Delay, State, Status
- [ ] - Thread control i.e. Stop thread, disable thread.
