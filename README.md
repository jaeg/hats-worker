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
- scripts - scripts to register  
- run-now - run registered scripts on this wart immediately

## Getting dependencies
- install deps
  - https://github.com/golang/dep
- install dependencies
  - `deps ensure`

## Get up and running
- Build it
  - `go build`
- You can get started using an example config as such
  -  `./redis-wart --config wart1.config`
- Or you can pass in through runtime params  
  - `./redis-wart --redis-address=<address> --redis-password=<password> --wart-name=wart1`
- Or run through a docker container
  - `docker run jaeg/redis-wart:latest --redis-address=<address> --redis-password=<password> --wart-name=wart1`

## Javascript implementation
Wart's Javascript implementation is based on [Otto](https://github.com/robertkrimen/otto).  Each thread maintains its own scope.  When a thread starts it runs the entire script.  It then runs `init()` if present in the source code.  If present a thread will call `main()` after confirming the thread is still running.

### Extra Javascript Functions
Otto only include what's in the base Javascript definition and lacks certain useful libraries.  Those we recreated are:

#### Wart
- wart.Name
  - returns string
- wart.Cluster
  - returns string
- wart.ShuttingDown - It is suggested that if you have code that loops you also check this to make sure the code end cleanly.
  - returns bool

#### Thread
- thread.Key
  - returns string
- thread.State() 
  - returns string
- thread.Status()
  - returns string
- thread.Stopped - It is suggested that if you have code that loops you also check this to make sure the code end cleanly.
  - returns bool
- thread.Disable() - Disables the thread completely
  - returns nothing
- thread.Stop() - Stops the thread causing another node to possibly pick it up
  - returns nothing
#### HTTP
- http.Get(url)
  - returns {body:'',headers:[], status: 200}
- http.Post(url, body)
  - returns {body:'',headers:[], status: 200}
- http.PostForm(url, bodyObject)
  - returns {body:'',headers:[], status: 200}
- http.Put(url, body)
  - returns {body:'',headers:[], status: 200}
- http.Head(url)
  - returns {headers:[], status: 200}
- http.Delete(url)
  - returns {body:'',headers:[], status: 200}

#### Redis
- redis.Do(method, args....)
  - return [response from redis, error (if one)]

#### Response
- response.Write(value)
  - Used when a wart is in endpoint mode.  Writes to the response body of an http request.
  - returns nothing.

#### Request
- request.Method
  - returns string
- request.Path
  - returns string
- request.Query
  - returns {<name>:[values]}
- request.Body
  - returns string

### Wart Todo
- [x] - Run a thread from redis.
- [x] - Create thread from file.
- [x] - Stop thread if wart is unhealthy.
- [x] - Stop thread if status is disabled.
- [x] - Stop thread if not the owner of thread.
- [x] - CPU health check based on threshold.
- [x] - Memory health check based on threshold.
- [x] - Recover from critical state when thresholds are met.

### Javascript Todo
- [x] - Basic javascript implementation
- [x] - Keep scope inside of thread
- [x] - Redis wrapper
- [x] - Http wrapper
- [x] - Wart information i.e. Health, name, cluster
- [x] - Thread information i.e. Delay, State, Status
- [x] - Thread control i.e. Stop thread, disable thread.
