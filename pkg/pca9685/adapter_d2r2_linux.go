//go:build linux

package pca9685

import (
	"fmt"
	"github.com/d2r2/go-i2c"
)

// I2CAdapterD2r2 оборачивает объект *i2c.I2C из библиотеки d2r2/go-i2c.
type I2CAdapterD2r2 struct {
	dev    *i2c.I2C
	logger Logger
}

// NewI2CAdapterD2r2 создаёт новый адаптер для d2r2/go-i2c.
func NewI2CAdapterD2r2(dev *i2c.I2C) *I2CAdapterD2r2 {
	return &I2CAdapterD2r2{
		dev:    dev,
		logger: NewDefaultLogger(LogLevelBasic),
	}
}

func (a *I2CAdapterD2r2) WriteReg(reg uint8, data []byte) error {
	a.logger.Detailed("I2CAdapterD2r2: WriteReg: register=0x%X, data=%v", reg, data)
	buf := append([]byte{reg}, data...)
	n, err := a.dev.WriteBytes(buf)
	if err != nil {
		a.logger.Error("I2CAdapterD2r2: WriteReg: error writing bytes: %v", err)
		return err
	}
	if n != len(buf) {
		err = fmt.Errorf("WriteReg: wrote %d bytes, expected %d", n, len(buf))
		a.logger.Error("I2CAdapterD2r2: WriteReg: %v", err)
		return err
	}
	a.logger.Detailed("I2CAdapterD2r2: WriteReg: success")
	return nil
}

func (a *I2CAdapterD2r2) ReadReg(reg uint8, data []byte) error {
	a.logger.Detailed("I2CAdapterD2r2: ReadReg: register=0x%X", reg)
	_, err := a.dev.WriteBytes([]byte{reg})
	if err != nil {
		a.logger.Error("I2CAdapterD2r2: ReadReg: error writing register: %v", err)
		return err
	}
	n, err := a.dev.ReadBytes(data)
	if err != nil {
		a.logger.Error("I2CAdapterD2r2: ReadReg: error reading bytes: %v", err)
		return err
	}
	if n != len(data) {
		err = fmt.Errorf("ReadReg: read %d bytes, expected %d", n, len(data))
		a.logger.Error("I2CAdapterD2r2: ReadReg: %v", err)
		return err
	}
	a.logger.Detailed("I2CAdapterD2r2: ReadReg: success, data=%v", data)
	return nil
}

func (a *I2CAdapterD2r2) Close() error {
	a.logger.Basic("I2CAdapterD2r2: Closing device")
	return a.dev.Close()
}
