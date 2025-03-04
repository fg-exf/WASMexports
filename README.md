# WASMexports

Looking into wasm I discovered that it had exports like malloc and free, which can be externally referenced and accessed by TinyGo. I first found out about them here: https://github.com/tinygo-org/tinygo/issues/2788. Digging deeper I realized that these exports can be used to pass data back and forth between the OS and wasm. Hopefully someone finds this PoC useful. 

installation: 

```git clone https://github.com/fg-exf/WASMexports.git```

```go get .```

compile the wasm binary: ```tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o buildUrl.wasm buildUrl.go```

build the main: ```go build main.go```

example usage to run main: ```./main --url api.ipify.org```
