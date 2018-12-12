function init() {
  console.log("Main init")
  var req = http.Post("https://httpbin.org/post","{'test':'Hi!'}")
  var body = JSON.parse(req.body)
  console.log(req.headers)
  for (var key in req.headers) {
    console.log(key)
  }
}

function main() {
}
