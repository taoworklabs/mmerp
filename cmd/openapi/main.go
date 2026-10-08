// Command openapi writes the API spec to stdout for make gen.
package main

import (
	"log"
	"os"

	"github.com/taoworklabs/mmerp/internal/app"
)

func main() {
	spec, err := app.OpenAPI()
	if err != nil {
		log.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(spec, '\n'))
}
