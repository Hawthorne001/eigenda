# VARIABLES
variable "REGISTRY" {
  default = "ghcr.io"
}

variable "REPOSITORY" {
  default = "layr-labs/eigenda"
}

variable "BUILD_TAG" {
  default = "latest"
}

variable "SEMVER" {
  default = "v0.0.0"
}

variable "GITCOMMIT" {
  default = "dev"
}

variable "GITDATE" {
  default = "0"
}

# GROUPS

group "default" {
  targets = ["all"]
}

group "all" {
  targets = ["node-group", "disperser-group", "retriever", "churner"]
}

group "node-group" {
  targets = ["node", "nodeplugin"]
}

group "disperser-group" {
  targets = ["batcher", "disperser", "encoder"]
}

group "node-group-release" {
  targets = ["node-release", "nodeplugin-release"]
}

# DISPERSER TARGETS

target "batcher" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "batcher"
  tags       = ["${REGISTRY}/${REPOSITORY}/batcher:${BUILD_TAG}"]
}

target "disperser" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "apiserver"
  tags       = ["${REGISTRY}/${REPOSITORY}/disperser:${BUILD_TAG}"]
}

target "encoder" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "encoder"
  tags       = ["${REGISTRY}/${REPOSITORY}/encoder:${BUILD_TAG}"]
}

target "retriever" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "retriever"
  tags       = ["${REGISTRY}/${REPOSITORY}/retriever:${BUILD_TAG}"]
}

target "churner" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "churner"
  tags       = ["${REGISTRY}/${REPOSITORY}/churner:${BUILD_TAG}"]
}

# NODE TARGETS

target "node" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "node"
  tags       = ["${REGISTRY}/${REPOSITORY}/node:${BUILD_TAG}"]
  args = {
    SEMVER    = "${SEMVER}"
    GITCOMMIT = "${GITCOMMIT}"
    GITDATE   = "${GITDATE}"
  }
}

target "nodeplugin" {
  context    = "."
  dockerfile = "./Dockerfile"
  target     = "nodeplugin"
  tags       = ["${REGISTRY}/${REPOSITORY}/nodeplugin:${BUILD_TAG}"]
}

# RELEASE TARGETS

target "_release" {
  platforms = ["linux/amd64", "linux/arm64"]
}

target "node-release" {
  inherits = ["node", "_release"]
  tags     = ["${REGISTRY}/${REPOSITORY}/opr-node:${BUILD_TAG}"]
}

target "nodeplugin-release" {
  inherits = ["nodeplugin", "_release"]
  tags     = ["${REGISTRY}/${REPOSITORY}/opr-nodeplugin:${BUILD_TAG}"]
}