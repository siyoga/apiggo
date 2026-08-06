// Package getpet implements the GetPet operation handler for the petstore example.
package getpet

import (
	"context"

	dto "github.com/siyoga/apiggo/examples/petstore/generated/dto"
	"github.com/siyoga/apiggo/examples/petstore/store"
)

// Handler implements the GetPet operation.
type Handler struct {
	pets *store.Store
}

// New builds a Handler. Add your dependencies here.
func New(pets *store.Store) *Handler {
	return &Handler{pets: pets}
}

// Handle implements the GetPet operation (GET /pets/{id}).
//
// Returning *dto.GetPetNotFound (a generated APIError) makes the runtime reply
// with 404 and the error's JSON body — no status juggling in the handler.
func (h *Handler) Handle(_ context.Context, in dto.GetPetIn) (dto.GetPetOut, error) {
	pet, ok := h.pets.Get(in.Id)
	if !ok {
		msg := "pet not found"
		return dto.GetPetOut{}, &dto.GetPetNotFound{Message: &msg}
	}
	return pet, nil
}
