GO := "go"
BINARY_NAME := "markdown-gokil"

default: build

build:
    {{GO}} build -o build/{{BINARY_NAME}} cmd/{{BINARY_NAME}}/main.go

clean:
    rm -rf build/
    rm -rf outputs/
    rm -f {{BINARY_NAME}}


run input output="":
    {{GO}} run cmd/{{BINARY_NAME}}/main.go {{input}} {{output}}

mcp:
    {{GO}} run cmd/{{BINARY_NAME}}/main.go -mcp

tidy:
    {{GO}} mod tidy
