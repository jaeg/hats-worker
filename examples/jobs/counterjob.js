console.log("Counter Job running")
redis.Do("incr","CounterJob")

var value = redis.Do("get","CounterJob")
console.log(value)