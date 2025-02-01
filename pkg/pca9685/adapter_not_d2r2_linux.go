//go:build windows || darwin
// +build windows darwin

package pca9685

import (
	"fmt"
)

// NewI2CAdapterD2r2 выводит сообщение об ошибке и возвращает ошибку.
func NewI2CAdapterD2r2() error {
	return fmt.Errorf("ПРЕДУПРЕЖДЕНИЕ: адаптер d2r2/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.")
}
