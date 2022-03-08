target "docker-metadata-action" {}

target "build" {
  inherits = ["docker-metadata-action"]
  context = "./"
  platforms = [
    "linux/386",
    "linux/amd64",
    "linux/arm",
    "linux/arm64",
    "linux/mips",
    "linux/mips64",
    "linux/mips64le",
    "linux/mipsle",
    "linux/ppc64",
    "linux/ppc64le",
    "linux/riscv64",
    "linux/s390x",
  ]
}
