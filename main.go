package main

import (
	"log"

	"github.com/isaschm/admission-controller-webhook-demo/cmd/webhookserver"
)

func main() {
	if err := webhookserver.ExecuteServe(); err != nil {
		log.Fatal(err)
	}
}
