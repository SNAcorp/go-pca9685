//go:build !linux

package pca9685

import "fmt"

// ПРЕДУПРЕЖДЕНИЕ: адаптер periph.io/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.
func NewI2CAdapterPeriph() error {
	return fmt.Errorf("ПРЕДУПРЕЖДЕНИЕ: адаптер periph.io/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.")
}
