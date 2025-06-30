package service

import "github.com/google/uuid"

// TokenGenerator adalah interface untuk membuat token acak.
// Dengan adanya interface, kita bisa menggantinya dengan mock saat testing.
type TokenGenerator interface {
	Generate() string
}

// UUIDTokenGenerator adalah implementasi nyata yang akan digunakan di produksi.
type UUIDTokenGenerator struct{}

// Generate membuat token acak menggunakan UUID v4.
func (g *UUIDTokenGenerator) Generate() string {
	return uuid.NewString()
}
