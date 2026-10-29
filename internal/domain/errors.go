package domain

import "errors"

// Sentinel domain errors.
var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrUnsupportedPlatform    = errors.New("unsupported platform for this operation")
	ErrConflict               = errors.New("configuration conflict detected")
	ErrTransactionInterrupted = errors.New("previous transaction was interrupted")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrBackupNotFound         = errors.New("backup snapshot not found")
	ErrQuotaExceeded          = errors.New("backup disk quota exceeded")
	ErrValidationFailed       = errors.New("manifest or configuration validation failed")
	ErrProviderUnavailable    = errors.New("package provider is unavailable on this system")
)
