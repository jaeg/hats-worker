# redis-wart
A simple interpreter designed to process data sitting in redis.

A wart is a single server instance running any number of scripts.  It connects directly to redis and
provides status updates as to it's health.  If its health is critical it puts scripts that it is running into redis.
From there another healthy wart can pick up the script and run it in its place.

Current Features:
Loads a list of files into a map.
Registers itself with redis.
Checks health based on CPU load.
  -If over threshold goes into crit status.
  -If in crit status it figures out what to give up based on what scripts are running and puts them into redis.
Checks status of other warts.
  -If any are in crit status and it is not in crit status it checks jobs.


  Setup Instructions:
  Clone
  Run `go get`
  Run `go run *.go --redis-address=<address> --redis-password=<password --scripts=examples/main.txt,examples/hello.txt --max-cpu=2 --wart-name=wart1`
