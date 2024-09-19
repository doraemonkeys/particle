#!/bin/bash

# PreInstall
# go install github.com/doraemonkeys/gobuild@latest

# windows amd64
gobuild -t windows -o particle.exe
zip -r particle-windows-amd64.zip particle.exe

# windows arm64
gobuild -t windows -arch arm64 -o particle.exe
zip -r particle-windows-arm64.zip particle.exe

# linux amd64
gobuild -t linux -o particle
zip -r particle-linux-amd64.zip particle

# linux arm64
gobuild -t linux -arch arm64 -o particle
zip -r particle-linux-arm64.zip particle

# mac amd64
gobuild -t darwin -o particle
zip -r particle-mac-amd64.zip particle

# mac arm64
gobuild -t darwin -arch arm64 -o particle
zip -r particle-mac-arm64.zip particle
