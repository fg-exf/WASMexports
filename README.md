# WASMexports

Looking into wasm I discovered that it had exports like malloc and free, which can be externally referenced and accessed by TinyGo. I first found out about them here: https://github.com/tinygo-org/tinygo/issues/2788. Digging deeper I realized that these exports can be used to pass data back and forth between the OS and wasm. Hopefully someone finds this PoC useful. 

It leverages the cobra library to parse command line arguments of hostnames and perform IP lookups of them, by passing the domain name to wasm, passing it back to TinyGo via the malloc exports to perform the DNS lookup, then passes it the IP address back to wasm via the malloc export again. It seems overly complicated, because it is. It's just meant to be an example of how the malloc and free exports work. I've commented the code as much as possible to make it easy to follow and replicate. 


how to compile the wasm binary: ```tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o buildUrl.wasm buildUrl.go```


example usage to run main: ```./main --url api.ipify.org```
