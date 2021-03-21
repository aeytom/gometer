#!/bin/bash

set -o pipefail -o nounset -o errexit

cd $(dirname $0)

export GOARCH="arm"
export GOHOSTARCH="arm"
export GOHOSTOS="linux"
export GOOS="linux"

go build 