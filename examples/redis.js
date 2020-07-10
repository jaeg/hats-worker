
  console.log("Redis example")
  redis.Do("set","RedisExample","Hi")

  var value = redis.Do("get","RedisExample")
  console.log(value)


function main() {

}
