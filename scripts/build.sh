#!/usr/bin/env bash
set -exuo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc

GOOS=windows go build -ldflags="-s -w" -o bin/supply.exe dynatrace-hwc-extension/supply/cli
GOOS=windows go build -ldflags="-s -w" -o bin/supply.exe dynatrace-hwc-extension/release/cli
GOOS=windows go build -ldflags="-s -w" -o bin/finalize.exe dynatrace-hwc-extension/finalize/cli
#GOOS=windows go build -ldflags="-s -w" -o bin/finalize.exe /Users/asad.ali/dT/specialProjects/dynatrace-dotnet-buildback-tile/hwc-extension/src/dynatrace-hwc-extension/finalize/cli

