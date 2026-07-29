package getpet

import (
	"context"

	dto "example.com/svc/generated/dto"
)

// Handler implements the GetPet operation.
type Handler struct{}

// New builds a Handler. Add your dependencies here.
func New() *Handler {
	return &Handler{}
}

// Handle implements the GetPet operation (GET /pets/{id}).
func (h *Handler) Handle(ctx context.Context, in dto.GetPetIn) (dto.GetPetOut, error) {
	// TODO: implement GetPet
	var out dto.GetPetOut
	return out, nil
}
