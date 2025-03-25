#!/bin/bash

# Build the GoGDBLLM application
go build -o gogdbllm ./cmd/gogdbllm

# Ensure uploads directory exists
mkdir -p uploads

# Run the application
./gogdbllm 