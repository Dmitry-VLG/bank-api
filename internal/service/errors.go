package service

import "errors"

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrDataIntegrity      = errors.New("card data integrity check failed")
)
