package pca9685

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	// Регистры MODE1
	RegMode1     = 0x00
	Mode1Sleep   = 0x10
	Mode1AutoInc = 0x20
	Mode1Restart = 0x80
	Mode1AllCall = 0x01

	// Регистры MODE2
	RegMode2    = 0x01
	Mode2OutDrv = 0x04
	Mode2Invrt  = 0x10
	Mode2OutNe  = 0x01

	// Регистры LED
	RegLed0     = 0x06
	RegAllLed   = 0xFA
	RegPrescale = 0xFE

	// Константы
	PwmResolution = 4096
	MinFrequency  = 24
	MaxFrequency  = 1526
	OscClock      = 25000000 // 25MHz
)

// I2C определяет интерфейс для работы с I2C устройством
type I2C interface {
	WriteReg(reg uint8, data []byte) error
	ReadReg(reg uint8, data []byte) error
	Close() error
}

// Channel представляет один PWM канал
type Channel struct {
	mu      sync.RWMutex
	enabled bool
	on      uint16
	off     uint16
}

// PCA9685 представляет контроллер PWM
type PCA9685 struct {
	dev      I2C
	mu       sync.RWMutex
	Freq     float64
	channels [16]Channel
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config содержит настройки для инициализации PCA9685
type Config struct {
	InitialFreq float64         // Начальная частота PWM (24-1526 Hz)
	InvertLogic bool            // Инвертировать выходную логику
	OpenDrain   bool            // Использовать open-drain выходы
	Context     context.Context // Контекст для отмены операций
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		InitialFreq: 1000,
		InvertLogic: false,
		OpenDrain:   false,
		Context:     context.Background(),
	}
}

// New создает новый экземпляр PCA9685 с указанной конфигурацией
func New(dev I2C, config *Config) (*PCA9685, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(config.Context)
	pca := &PCA9685{
		dev:    dev,
		ctx:    ctx,
		cancel: cancel,
	}

	// Инициализация каналов
	for i := range pca.channels {
		pca.channels[i].enabled = true
	}

	if err := pca.Reset(); err != nil {
		return nil, fmt.Errorf("failed to reset device: %w", err)
	}

	// Настройка MODE2
	mode2 := byte(0)
	if !config.OpenDrain {
		mode2 |= Mode2OutDrv
	}
	if config.InvertLogic {
		mode2 |= Mode2Invrt
	}
	if err := pca.dev.WriteReg(RegMode2, []byte{mode2}); err != nil {
		return nil, fmt.Errorf("failed to configure MODE2: %w", err)
	}

	// Установка частоты
	if err := pca.SetPWMFreq(config.InitialFreq); err != nil {
		return nil, fmt.Errorf("failed to set frequency: %w", err)
	}

	return pca, nil
}

// Close освобождает ресурсы и закрывает устройство
func (pca *PCA9685) Close() error {
	pca.cancel()
	return pca.dev.Close()
}

// Reset инициализирует устройство с настройками по умолчанию
func (pca *PCA9685) Reset() error {
	pca.mu.Lock()
	defer pca.mu.Unlock()

	if err := pca.dev.WriteReg(RegMode1, []byte{Mode1Sleep | Mode1AutoInc}); err != nil {
		return fmt.Errorf("failed to set MODE1: %w", err)
	}
	return nil
}

// SetPWMFreq устанавливает частоту PWM в герцах (24-1526 Гц)
func (pca *PCA9685) SetPWMFreq(freq float64) error {
	if freq < MinFrequency || freq > MaxFrequency {
		return fmt.Errorf("frequency out of range (%d-%d Hz)", MinFrequency, MaxFrequency)
	}

	pca.mu.Lock()
	defer pca.mu.Unlock()

	// Вычисление значения предделителя
	prescale := math.Round(float64(OscClock)/(float64(PwmResolution)*freq)) - 1
	if prescale < 3 {
		prescale = 3
	}

	// Чтение текущего режима
	oldMode, err := pca.readMode1()
	if err != nil {
		return fmt.Errorf("failed to read MODE1: %w", err)
	}

	// Перевод в режим сна для изменения частоты
	if err := pca.dev.WriteReg(RegMode1, []byte{(oldMode & 0x7F) | Mode1Sleep}); err != nil {
		return fmt.Errorf("failed to enter sleep mode: %w", err)
	}

	// Установка предделителя
	if err := pca.dev.WriteReg(RegPrescale, []byte{byte(prescale)}); err != nil {
		return fmt.Errorf("failed to set prescale: %w", err)
	}

	// Восстановление предыдущего режима
	if err := pca.dev.WriteReg(RegMode1, []byte{oldMode}); err != nil {
		return fmt.Errorf("failed to restore mode: %w", err)
	}

	// Ожидание стабилизации осциллятора
	time.Sleep(500 * time.Microsecond)

	// Включение автоинкремента и рестарта
	if err := pca.dev.WriteReg(RegMode1, []byte{oldMode | Mode1Restart | Mode1AutoInc}); err != nil {
		return fmt.Errorf("failed to enable auto-increment: %w", err)
	}

	pca.Freq = freq
	return nil
}

// SetPWM устанавливает значения PWM для канала
func (pca *PCA9685) SetPWM(ctx context.Context, channel int, on, off uint16) error {
	if err := pca.validateChannel(channel); err != nil {
		return err
	}

	ch := &pca.channels[channel]
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.enabled {
		return fmt.Errorf("channel %d is disabled", channel)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		baseReg := uint8(RegLed0 + 4*channel)
		data := []byte{
			byte(on & 0xFF),
			byte(on >> 8),
			byte(off & 0xFF),
			byte(off >> 8),
		}
		if err := pca.dev.WriteReg(baseReg, data); err != nil {
			return fmt.Errorf("failed to set PWM values: %w", err)
		}

		ch.on = on
		ch.off = off
		return nil
	}
}

// SetAllPWM устанавливает одинаковые значения PWM для всех каналов
func (pca *PCA9685) SetAllPWM(ctx context.Context, on, off uint16) error {
	pca.mu.Lock()
	defer pca.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		data := []byte{
			byte(on & 0xFF),
			byte(on >> 8),
			byte(off & 0xFF),
			byte(off >> 8),
		}
		if err := pca.dev.WriteReg(RegAllLed, data); err != nil {
			return fmt.Errorf("failed to set all PWM values: %w", err)
		}

		for i := range pca.channels {
			if pca.channels[i].enabled {
				pca.channels[i].on = on
				pca.channels[i].off = off
			}
		}
		return nil
	}
}

// SetMultiPWM устанавливает разные значения PWM для нескольких каналов
func (pca *PCA9685) SetMultiPWM(ctx context.Context, settings map[int]struct{ On, Off uint16 }) error {
	// Проверка всех каналов перед установкой
	for channel := range settings {
		if err := pca.validateChannel(channel); err != nil {
			return err
		}
	}

	for channel, values := range settings {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := pca.SetPWM(ctx, channel, values.On, values.Off); err != nil {
				return fmt.Errorf("failed to set PWM for channel %d: %w", channel, err)
			}
		}
	}
	return nil
}

// EnableChannels включает указанные каналы
func (pca *PCA9685) EnableChannels(channels ...int) error {
	for _, ch := range channels {
		if err := pca.validateChannel(ch); err != nil {
			return err
		}
		pca.channels[ch].mu.Lock()
		pca.channels[ch].enabled = true
		pca.channels[ch].mu.Unlock()
	}
	return nil
}

// DisableChannels выключает указанные каналы
func (pca *PCA9685) DisableChannels(channels ...int) error {
	for _, ch := range channels {
		if err := pca.validateChannel(ch); err != nil {
			return err
		}
		pca.channels[ch].mu.Lock()
		pca.channels[ch].enabled = false
		// Установка нулевого выхода для выключенного канала
		if err := pca.SetPWM(pca.ctx, ch, 0, 0); err != nil {
			pca.channels[ch].mu.Unlock()
			return fmt.Errorf("failed to disable channel %d: %w", ch, err)
		}
		pca.channels[ch].mu.Unlock()
	}
	return nil
}

// GetChannelState возвращает текущее состояние канала
func (pca *PCA9685) GetChannelState(channel int) (enabled bool, on, off uint16, err error) {
	if err := pca.validateChannel(channel); err != nil {
		return false, 0, 0, err
	}

	ch := &pca.channels[channel]
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.enabled, ch.on, ch.off, nil
}

// validateChannel проверяет корректность номера канала
func (pca *PCA9685) validateChannel(channel int) error {
	if channel < 0 || channel > 15 {
		return fmt.Errorf("invalid channel number: %d", channel)
	}
	return nil
}

// readMode1 читает текущее значение регистра MODE1
func (pca *PCA9685) readMode1() (byte, error) {
	data := make([]byte, 1)
	if err := pca.dev.ReadReg(RegMode1, data); err != nil {
		return 0, fmt.Errorf("failed to read MODE1: %w", err)
	}
	return data[0], nil
}
