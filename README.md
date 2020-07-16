# hats-worker
**H** - Highly  
**A** - Available  
**T** - Tasking    
**S** - System  
A simple interpreter designed to process data sitting in redis.

Each worker in a cluster checks in redis for work to do.  If it finds a stopped thread or a dead thread it takes the thread and runs it locally. There are also jobs which instead of continuously running they execute on a cron schedule.

## Runtime params
- cluster-name - name of cluster   
- worker-name - name of the worker   
- redis-address - address to redis server  
- redis-password - password for redis server   
- scripts - scripts to register  
- run-now - run registered scripts on this worker immediately

## Getting dependencies
Requires a version of go that supports go.mod
- go get

## Get up and running
- Build it
  - `make build`
- Build and run it
  - `make run`
- You can get started using an example config as such
  -  `./bin/worker --config worker1.config`
- Or you can pass in through runtime params  
  - `./bin/worker --redis-address=<address> --redis-password=<password> --worker-name=worker1`
- Or run through a docker container
  - `docker run jaeg/hats-worker:latest --redis-address=<address> --redis-password=<password> --worker-name=worker1`
- You can also tell it to load a script on start like this
  - `./bin/worker --redis-address=<address> --redis-password=<password> --worker-name=worker1 --scripts examples/hello.js`
  - `docker run jaeg/hats-worker:latest --redis-address=<address> --redis-password=<password> --worker-name=worker1 --scripts examples/hello.js`

## Javascript implementation
worker's Javascript implementation is based on [Otto](https://github.com/robertkrimen/otto).  Each thread maintains its own scope.  When a thread starts it runs the entire script.  It then runs `init()` if present in the source code.  If present a thread will call `main()` after confirming the thread is still running.

### Extra Functions Available to scripts
#### Env
- env.Get(key)
  - returns string or undefined if key does not exist
- env.Set(key,value)
  - returns null or error if there is one
- env.Unset(key)
  - returns null or error if there is one


#### worker
- worker.Name
  - returns string
- worker.Cluster
  - returns string
- worker.ShuttingDown - It is suggested that if you have code that loops you also check this to make sure the code end cleanly.
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

#### Job
- job.Key
  - returns string
- job.State() 
  - returns string
- thread.Status()
  - returns string
- thread.Stopped - It is suggested that if you have code that loops you also check this to make sure the code end cleanly.
  - returns bool
- thread.Disable() - Disables the thread completely
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
  - Used when a worker is in endpoint mode.  Writes to the response body of an http request.
  - returns nothing.
- response.Error(errorString, statusCode)
  - Used when a worker is in endpoint mode.  Writes an error in response to the endpoint.
  - returns nothing.  
- response.SetContentType(type)
  - Used when a worker is in endpoint mode.  Sets the content type header.
  - returns nothing.  
- response.SetHeader(key,value)
  - Used when a worker is in endpoint mode.  Sets the header specfied in key to the value.
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
- request.GetHeader(key)
  - Used when a worker is in endpoint mode.  Gets value from header.
  - returns value.  

#### SQL
- sql.New(connectionString, driverType)
  - supported driver types: mysql
  - returns db interface
- db.Ping()
  - returns an error if it fails to ping the db
- db.Query(query, arguments...)
  - returns rows or nil if error
- db.Exec(query, arguments...)
  - returns number of impacted rows or undefined if error
- db.Close()
  - returns error if any
 

### worker Todo
- [x] - Run a thread from redis.
- [x] - Create thread from file.
- [x] - Stop thread if worker is unhealthy.
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
- [x] - worker information i.e. Health, name, cluster
- [x] - Thread information i.e. Delay, State, Status
- [x] - Thread control i.e. Stop thread, disable thread.
