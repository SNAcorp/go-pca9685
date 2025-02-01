//go:build windows || darwin
// +build windows darwin

package pca9685

// NewI2CAdapterPeriph выводит сообщение об ошибке и возвращает ошибку.
func NewI2CAdapterPeriph() error {
	return fmt.Errorf("ПРЕДУПРЕЖДЕНИЕ: адаптер periph.io/go-i2c работает только на Linux. Используйте тестовый адаптер для вашей системы.")
}
