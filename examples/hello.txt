var count = 0

function main() {
  count += 1
  console.log("Hello " + count + " many times.")
}

function cleanup() {
  count = 0
  console.log("Clean up complete")
}