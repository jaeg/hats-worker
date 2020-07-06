
function main() {
  console.log("State: ", thread.State())
  console.log("Status: ", thread.Status())
  console.log("Stopped: ", thread.Stopped)
  thread.Disable()
  console.log("Status: ", thread.Status())
  console.log("Stopped: ", thread.Stopped)
}

function cleanup() {
}