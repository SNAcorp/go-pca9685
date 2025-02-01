//go:build !linux

package pca9685

import (
	"fmt"
)

// ПРЕДУПРЕЖДЕНИЕ: адаптер d2r2/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.
func NewI2CAdapterD2r2() error {
	return fmt.Errorf("ПРЕДУПРЕЖДЕНИЕ: адаптер d2r2/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.")
}
