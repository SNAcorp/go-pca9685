//go:build linux

package pca9685

import (
	"fmt"

	"github.com/d2r2/go-i2c"
)

// I2CAdapterD2r2 оборачивает объект *i2c.I2C из библиотеки d2r2/go-i2c.
type I2CAdapterD2r2 struct {
	dev *i2c.I2C
}

// NewI2CAdapterD2r2 создаёт новый адаптер для d2r2/go-i2c.
func NewI2CAdapterD2r2(dev *i2c.I2C) *I2CAdapterD2r2 {
	return &I2CAdapterD2r2{dev: dev}
}

func (a *I2CAdapterD2r2) WriteReg(reg uint8, data []byte) error {
	buf := append([]byte{reg}, data...)
	n, err := a.dev.WriteBytes(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("WriteReg: wrote %d bytes, expected %d", n, len(buf))
	}
	return nil
}

func (a *I2CAdapterD2r2) ReadReg(reg uint8, data []byte) error {
	_, err := a.dev.WriteBytes([]byte{reg})
	if err != nil {
		return err
	}
	n, err := a.dev.ReadBytes(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("ReadReg: read %d bytes, expected %d", n, len(data))
	}
	return nil
}

func (a *I2CAdapterD2r2) Close() error {
	return a.dev.Close()
}
