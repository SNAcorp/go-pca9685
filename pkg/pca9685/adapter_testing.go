package pca9685

import "sync"

///////////////////////////////////////////////////////////////////////////////
// Адаптер TestI2C (эмуляция I2C для MacOS/Windows или тестового устройства)
///////////////////////////////////////////////////////////////////////////////

type TestI2C struct {
	mu        sync.RWMutex
	registers map[uint8][]byte
}

// NewTestI2C создаёт новый адаптер-эмулятор I2C.
func NewTestI2C() *TestI2C {
	return &TestI2C{
		registers: make(map[uint8][]byte),
	}
}

// WriteReg эмулирует запись в регистр, сохраняя данные в памяти.
func (t *TestI2C) WriteReg(reg uint8, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	buf := make([]byte, len(data))
	copy(buf, data)
	t.registers[reg] = buf
	return nil
}

// ReadReg эмулирует чтение из регистра. Если значение не найдено, возвращает нули.
func (t *TestI2C) ReadReg(reg uint8, data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if buf, ok := t.registers[reg]; ok {
		n := copy(data, buf)
		// Если записанное значение короче запрошенного, дополняем нулями.
		for i := n; i < len(data); i++ {
			data[i] = 0
		}
		return nil
	}
	// Если регистр не найден, возвращаем нули.
	for i := range data {
		data[i] = 0
	}
	return nil
}

// Close эмулирует закрытие устройства (ничего не делает).
func (t *TestI2C) Close() error {
	return nil
}
