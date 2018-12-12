function init() {
  console.log("Redis example")
  redis.Do("set","RedisExample","Hi")

  var value = redis.Do("get","RedisExample")
  console.log(value.String()[0])
}

function main() {

}
