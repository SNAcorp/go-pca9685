//go:build linux

package pca9685

import (
	periph_i2c "periph.io/x/conn/v3/i2c"
)

// I2CAdapterPeriph реализует работу с I2C через periph.io.
type I2CAdapterPeriph struct {
	dev    *periph_i2c.Dev
	logger Logger
}

// NewI2CAdapterPeriph создаёт новый адаптер для periph.io.
func NewI2CAdapterPeriph(dev *periph_i2c.Dev) *I2CAdapterPeriph {
	return &I2CAdapterPeriph{
		dev:    dev,
		logger: NewDefaultLogger(LogLevelBasic),
	}
}

func (a *I2CAdapterPeriph) WriteReg(reg uint8, data []byte) error {
	a.logger.Detailed("I2CAdapterPeriph: WriteReg: register=0x%X, data=%v", reg, data)
	buf := append([]byte{reg}, data...)
	if err := a.dev.Tx(buf, nil); err != nil {
		a.logger.Error("I2CAdapterPeriph: WriteReg: error during Tx: %v", err)
		return err
	}
	a.logger.Detailed("I2CAdapterPeriph: WriteReg: success")
	return nil
}

func (a *I2CAdapterPeriph) ReadReg(reg uint8, data []byte) error {
	a.logger.Detailed("I2CAdapterPeriph: ReadReg: register=0x%X", reg)
	if err := a.dev.Tx([]byte{reg}, data); err != nil {
		a.logger.Error("I2CAdapterPeriph: ReadReg: error during Tx: %v", err)
		return err
	}
	a.logger.Detailed("I2CAdapterPeriph: ReadReg: success, data=%v", data)
	return nil
}

func (a *I2CAdapterPeriph) Close() error {
	a.logger.Basic("I2CAdapterPeriph: Close called")
	// Для periph.io обычно закрывать устройство не требуется.
	return nil
}
