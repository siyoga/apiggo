// Command petstore is the apiggo end-to-end example: an OpenAPI contract
// (api/openapi.yaml) generated into DTOs + HTTP adapters (generated/) with
// handler stubs (api/) implemented against an in-memory store, all wired onto
// the apiggo runtime and served over HTTP.
//
// Run it from this directory:
//
//	go run .
//
// Then, in another terminal:
//
//	curl -s -XPOST localhost:8080/pets -d '{"name":"Rex","status":"available"}'
//	curl -s localhost:8080/pets/1
//	curl -s -i localhost:8080/pets/999   # 404 from *dto.GetPetNotFound
//	curl -s -i -XPOST localhost:8080/pets -d '{}'  # 400 from *dto.CreatePetBadRequest
//
// Regenerate the code layers after editing the contract, from the repo root:
//
//	go run ./cmd/apiggo -config examples/petstore/apiggo.yaml
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/siyoga/apiggo/examples/petstore/api/createpet"
	"github.com/siyoga/apiggo/examples/petstore/api/getpet"
	"github.com/siyoga/apiggo/examples/petstore/generated/router"
	"github.com/siyoga/apiggo/examples/petstore/store"
	"github.com/siyoga/apiggo/server"
)

func main() {
	// One shared dependency, injected into each handler.
	pets := store.New()

	// WithOpenAPIMethod pairs a generated registrar (router.GetPet) with your
	// typed handler (getpet.New(pets).Handle); the runtime builds the HTTP
	// adapter and wires it onto the mux.
	srv := server.NewServer(
		server.WithOpenAPIMethod(router.GetPet, getpet.New(pets).Handle),
		server.WithOpenAPIMethod(router.CreatePet, createpet.New(pets).Handle),
	)

	// Serve blocks until SIGINT/SIGTERM, then gracefully shuts down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Println("petstore listening on :8080")
	if err := srv.Serve(ctx, ":8080"); err != nil {
		log.Fatal(err)
	}
}
