#!/bin/bash

if [ ! -e go.mod ]; then
  echo "go.mod file not found, Running: go mod init"
  go mod init
fi
echo "Running: go mod tidy"
go mod tidy
