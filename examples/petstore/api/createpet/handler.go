package createpet

import (
	"context"

	dto "github.com/siyoga/apiggo/examples/petstore/generated/dto"
	"github.com/siyoga/apiggo/examples/petstore/store"
)

// Handler implements the CreatePet operation.
type Handler struct {
	pets *store.Store
}

// New builds a Handler. Add your dependencies here.
func New(pets *store.Store) *Handler {
	return &Handler{pets: pets}
}

// Handle implements the CreatePet operation (POST /pets).
//
// The request body arrives already decoded in in.Body; validation failures are
// reported by returning *dto.CreatePetBadRequest (a generated 400 APIError).
func (h *Handler) Handle(ctx context.Context, in dto.CreatePetIn) (dto.CreatePetOut, error) {
	if in.Body.Name == "" {
		msg := "name is required"
		code := int32(1001)
		return dto.CreatePetOut{}, &dto.CreatePetBadRequest{Message: &msg, Code: &code}
	}

	status := dto.PetStatusAvailable
	if in.Body.Status != nil {
		status = dto.PetStatus(*in.Body.Status)
	}

	pet := h.pets.Create(dto.Pet{
		Name:   in.Body.Name,
		Status: status,
	})
	return pet, nil
}
