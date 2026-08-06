package createpet

import (
	"context"

	dto "example.com/svc/generated/dto"
)

// Handler implements the CreatePet operation.
type Handler struct{}

// New builds a Handler. Add your dependencies here.
func New() *Handler {
	return &Handler{}
}

// Handle implements the CreatePet operation (POST /pets).
func (h *Handler) Handle(ctx context.Context, in dto.CreatePetIn) (dto.CreatePetOut, error) {
	// TODO: implement CreatePet
	var out dto.CreatePetOut
	return out, nil
}
