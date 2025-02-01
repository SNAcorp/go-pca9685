//go:build linux
// +build linux

package pca9685

///////////////////////////////////////////////////////////////////////////////
// Адаптер для periph.io
///////////////////////////////////////////////////////////////////////////////

type I2CAdapterPeriph struct {
	dev *periph_i2c.Dev
}

func NewI2CAdapterPeriph(dev *periph_i2c.Dev) *I2CAdapterPeriph {
	return &I2CAdapterPeriph{dev: dev}
}

func (a *I2CAdapterPeriph) WriteReg(reg uint8, data []byte) error {
	buf := append([]byte{reg}, data...)
	if err := a.dev.Tx(buf, nil); err != nil {
		return err
	}
	return nil
}

func (a *I2CAdapterPeriph) ReadReg(reg uint8, data []byte) error {
	if err := a.dev.Tx([]byte{reg}, data); err != nil {
		return err
	}
	return nil
}

func (a *I2CAdapterPeriph) Close() error {
	// Для periph.io обычно закрывать устройство не требуется.
	return nil
}
