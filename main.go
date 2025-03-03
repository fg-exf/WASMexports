package main

import (
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// buildUrl.wasm | `tinygo build -o buildUrl.wasm -scheduler=none --no-debug -target=wasi buildUrl.go`
//
//go:embed buildUrl.wasm
var buildUrlWasm []byte

func main() {
	var url string

	var rootCmd = &cobra.Command{
		Use:   "http-request",
		Short: "HTTP Request CLI",
		Long:  `A simple CLI application to make HTTP GET requests and output the response.`,
		Run: func(cmd *cobra.Command, args []string) {
			if url == "" {
				fmt.Println("Please provide a URL using the --url flag.")
				return
			}
			// Choose the context to use for function calls.
			ctx := context.Background()

			// Create a new WebAssembly Runtime.
			r := wazero.NewRuntime(ctx)
			defer r.Close(ctx) // This closes everything this Runtime created.

			// Instantiate a Go-defined module named "env" that exports a function to
			// log to the console.
			_, err := r.NewHostModuleBuilder("env").NewFunctionBuilder().WithFunc(logString).Export("log").Instantiate(ctx)
			if err != nil {
				log.Panicln(err)
			}

			// Note: buildUrl.go doesn't use WASI, but TinyGo needs it to
			// implement functions such as panic.
			wasi_snapshot_preview1.MustInstantiate(ctx, r)

			// Instantiate a WebAssembly module that imports the "log" function defined
			// in "env" and exports "memory" and functions we'll use in this example.
			mod, err := r.InstantiateWithConfig(ctx, buildUrlWasm, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
			if err != nil {
				log.Panicln(err)
			}

			// Get references to WebAssembly functions
			message := mod.ExportedFunction("message")
			formatting := mod.ExportedFunction("formatting")

			// These are undocumented, but exported. See tinygo-org/tinygo#2788
			malloc := mod.ExportedFunction("malloc")
			free := mod.ExportedFunction("free")

			// Let's use the argument to this main function in Wasm.
			name := url
			nameSize := uint64(len(name))

			// Instead of an arbitrary memory offset, use TinyGo's allocator. Notice
			// there is nothing string-specific in this allocation function. The same
			// function could be used to pass binary serialized data to Wasm.
			results, err := malloc.Call(ctx, nameSize)
			if err != nil {
				log.Panicln(err)
			}
			namePtr := results[0]

			// This pointer is managed by TinyGo, but TinyGo is unaware of external usage.
			// So, we have to free it when finished
			defer free.Call(ctx, namePtr)

			// The pointer is a linear memory offset, which is where we write the name.
			if !mod.Memory().Write(uint32(namePtr), []byte(name)) {
				log.Panicf("Memory.Write(%d, %d) out of range of memory size %d",
					namePtr, nameSize, mod.Memory().Size())
			}

			// Now, we can call "message", which reads the string we wrote to memory!
			_, err = message.Call(ctx, namePtr, nameSize)
			if err != nil {
				log.Panicln(err)
			}

			// Finally, we get the formatted message "fomatting" printed. This shows how to
			// read-back something allocated by TinyGo.
			ptrSize, err := formatting.Call(ctx, namePtr, nameSize)
			if err != nil {
				log.Panicln(err)
			}

			formattingPtr := uint32(ptrSize[0] >> 32)
			formattingSize := uint32(ptrSize[0])

			// This pointer is managed by TinyGo, but TinyGo is unaware of external usage.
			// So, we have to free it when finished
			if formattingPtr != 0 {
				defer func() {
					_, err := free.Call(ctx, uint64(formattingPtr))
					if err != nil {
						log.Panicln(err)
					}
				}()
			}

			// The pointer is a linear memory offset, which is where we read the name.
			if bytes, ok := mod.Memory().Read(formattingPtr, formattingSize); !ok {
				log.Panicf("Memory.Read(%d, %d) out of range of memory size %d",
					formattingPtr, formattingSize, mod.Memory().Size())
			} else {
				fmt.Println("Requesting URL: ", string(bytes))
				resp, err := http.Get(string(bytes))
				if err != nil {
					fmt.Printf("Error making request: %v\n", err)
					return
				}
				defer resp.Body.Close()
				// Read the response body
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("Error reading response body: %v\n", err)
					return
				}

				// Output the response
				fmt.Println(string(body))
			}

		},
	}

	// Add a URL flag to the root command
	rootCmd.Flags().StringVarP(&url, "url", "u", "", "URL to make the HTTP GET request to")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func logString(_ context.Context, m api.Module, offset, byteCount uint32) {
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		log.Panicf("Memory.Read(%d, %d) out of range", offset, byteCount)
	}
	fmt.Println(string(buf))
}
