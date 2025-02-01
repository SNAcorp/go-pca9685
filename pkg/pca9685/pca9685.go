// pca9685.go
package pca9685

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// Основной код контроллера PCA9685
///////////////////////////////////////////////////////////////////////////////

const (
	// Регистр MODE1
	RegMode1     = 0x00
	Mode1Sleep   = 0x10
	Mode1AutoInc = 0x20
	Mode1Restart = 0x80
	Mode1AllCall = 0x01

	// Регистр MODE2
	RegMode2    = 0x01
	Mode2OutDrv = 0x04
	Mode2Invrt  = 0x10
	Mode2OutNe  = 0x01

	// Регистр для каналов LED
	RegLed0     = 0x06
	RegAllLed   = 0xFA
	RegPrescale = 0xFE

	// Константы
	PwmResolution = 4096
	MinFrequency  = 24
	MaxFrequency  = 1526
	OscClock      = 25000000 // 25 МГц
)

// I2C – минимальный интерфейс для работы с I²C устройствами.
type I2C interface {
	WriteReg(reg uint8, data []byte) error
	ReadReg(reg uint8, data []byte) error
	Close() error
}

// Channel представляет один PWM канал.
type Channel struct {
	mu      sync.RWMutex
	enabled bool
	on      uint16
	off     uint16
}

// PCA9685 представляет контроллер PCA9685.
type PCA9685 struct {
	dev      I2C
	mu       sync.RWMutex
	Freq     float64
	channels [16]Channel
	ctx      context.Context
	cancel   context.CancelFunc
	logger   Logger // добавлен логгер
}

// Config содержит настройки для инициализации PCA9685.
type Config struct {
	InitialFreq float64         // Начальная частота PWM (от 24 до 1526 Гц)
	InvertLogic bool            // Инвертировать выходную логику
	OpenDrain   bool            // Использовать open-drain выходы
	Context     context.Context // Контекст для отмены операций
	Logger      Logger          // Логгер. Если nil, будет использован стандартный.
	LogLevel    LogLevel        // Уровень логирования.
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		InitialFreq: 1000,
		InvertLogic: false,
		OpenDrain:   false,
		Context:     context.Background(),
		LogLevel:    LogLevelBasic,
		Logger:      NewDefaultLogger(LogLevelBasic),
	}
}

// New создаёт новый экземпляр PCA9685 с указанной конфигурацией.
func New(dev I2C, config *Config) (*PCA9685, error) {
	if config == nil {
		config = DefaultConfig()
	}
	// Если логгер не задан, используем дефолтный
	if config.Logger == nil {
		config.Logger = NewDefaultLogger(config.LogLevel)
	}

	ctx, cancel := context.WithCancel(config.Context)
	pca := &PCA9685{
		dev:    dev,
		ctx:    ctx,
		cancel: cancel,
		logger: config.Logger,
	}

	pca.logger.Basic("Создание экземпляра PCA9685, установка частоты: %v Гц", config.InitialFreq)

	// Инициализируем все каналы
	for i := range pca.channels {
		pca.channels[i].enabled = true
	}

	if err := pca.Reset(); err != nil {
		pca.logger.Error("Не удалось выполнить сброс устройства: %v", err)
		return nil, fmt.Errorf("failed to reset device: %w", err)
	}

	// Настройка регистра MODE2
	mode2 := byte(0)
	if !config.OpenDrain {
		mode2 |= Mode2OutDrv
	}
	if config.InvertLogic {
		mode2 |= Mode2Invrt
	}
	if err := pca.dev.WriteReg(RegMode2, []byte{mode2}); err != nil {
		pca.logger.Error("Не удалось настроить MODE2: %v", err)
		return nil, fmt.Errorf("failed to configure MODE2: %w", err)
	}
	pca.logger.Detailed("MODE2 установлен: 0x%X", mode2)

	// Установка частоты PWM
	if err := pca.SetPWMFreq(config.InitialFreq); err != nil {
		pca.logger.Error("Не удалось установить частоту: %v", err)
		return nil, fmt.Errorf("failed to set frequency: %w", err)
	}

	return pca, nil
}

// Close освобождает ресурсы и закрывает устройство.
func (pca *PCA9685) Close() error {
	pca.logger.Basic("Закрытие устройства")
	pca.cancel()
	return pca.dev.Close()
}

// EnableAllCall включает режим All Call.
func (pca *PCA9685) EnableAllCall() error {
	pca.logger.Detailed("Включение режима All Call")
	mode1, err := pca.readMode1()
	if err != nil {
		pca.logger.Error("Ошибка чтения MODE1: %v", err)
		return err
	}
	return pca.dev.WriteReg(RegMode1, []byte{mode1 | Mode1AllCall})
}

// Reset инициализирует устройство с настройками по умолчанию.
func (pca *PCA9685) Reset() error {
	pca.logger.Basic("Сброс устройства")
	pca.mu.Lock()
	defer pca.mu.Unlock()

	if err := pca.dev.WriteReg(RegMode1, []byte{Mode1Sleep | Mode1AutoInc}); err != nil {
		pca.logger.Error("Ошибка при установке MODE1: %v", err)
		return fmt.Errorf("failed to set MODE1: %w", err)
	}
	return nil
}

// SetPWMFreq устанавливает частоту PWM в герцах (от 24 до 1526 Гц).
func (pca *PCA9685) SetPWMFreq(freq float64) error {
	pca.logger.Basic("Установка частоты PWM: %v Гц", freq)
	if freq < MinFrequency || freq > MaxFrequency {
		err := fmt.Errorf("frequency out of range (%d-%d Hz)", MinFrequency, MaxFrequency)
		pca.logger.Error("Ошибка установки частоты: %v", err)
		return err
	}

	pca.mu.Lock()
	defer pca.mu.Unlock()

	// Вычисляем значение предделителя.
	prescale := math.Round(float64(OscClock)/(float64(PwmResolution)*freq)) - 1
	if prescale < 3 {
		prescale = 3
	}
	pca.logger.Detailed("Вычислен prescale: %v", prescale)

	// Чтение текущего режима.
	oldMode, err := pca.readMode1()
	if err != nil {
		pca.logger.Error("Ошибка чтения MODE1: %v", err)
		return fmt.Errorf("failed to read MODE1: %w", err)
	}

	// Переводим устройство в режим сна для установки предделителя.
	if err := pca.dev.WriteReg(RegMode1, []byte{(oldMode & 0x7F) | Mode1Sleep}); err != nil {
		pca.logger.Error("Не удалось войти в режим сна: %v", err)
		return fmt.Errorf("failed to enter sleep mode: %w", err)
	}

	// Записываем предделитель.
	if err := pca.dev.WriteReg(RegPrescale, []byte{byte(prescale)}); err != nil {
		pca.logger.Error("Не удалось установить prescale: %v", err)
		return fmt.Errorf("failed to set prescale: %w", err)
	}

	// Восстанавливаем прежний режим.
	if err := pca.dev.WriteReg(RegMode1, []byte{oldMode}); err != nil {
		pca.logger.Error("Не удалось восстановить режим: %v", err)
		return fmt.Errorf("failed to restore mode: %w", err)
	}

	// Короткая задержка для стабилизации осциллятора.
	time.Sleep(500 * time.Microsecond)

	// Включаем автоинкремент и рестарт.
	if err := pca.dev.WriteReg(RegMode1, []byte{oldMode | Mode1Restart | Mode1AutoInc}); err != nil {
		pca.logger.Error("Не удалось включить автоинкремент: %v", err)
		return fmt.Errorf("failed to enable auto-increment: %w", err)
	}

	pca.Freq = freq
	pca.logger.Detailed("Частота успешно установлена: %v Гц", pca.Freq)
	return nil
}

// SetPWM устанавливает значения PWM для указанного канала.
func (pca *PCA9685) SetPWM(ctx context.Context, channel int, on, off uint16) error {
	pca.logger.Detailed("SetPWM: канал %d, on=%d, off=%d", channel, on, off)
	if err := pca.validateChannel(channel); err != nil {
		pca.logger.Error("SetPWM: неверный номер канала %d: %v", channel, err)
		return err
	}

	ch := &pca.channels[channel]
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.enabled {
		err := fmt.Errorf("channel %d is disabled", channel)
		pca.logger.Error("SetPWM: канал отключён: %v", err)
		return err
	}

	select {
	case <-ctx.Done():
		err := ctx.Err()
		pca.logger.Error("SetPWM: контекст отменён: %v", err)
		return err
	default:
		baseReg := uint8(RegLed0 + 4*channel)
		data := []byte{
			byte(on & 0xFF),
			byte(on >> 8),
			byte(off & 0xFF),
			byte(off >> 8),
		}
		if err := pca.dev.WriteReg(baseReg, data); err != nil {
			pca.logger.Error("SetPWM: не удалось установить значения PWM: %v", err)
			return fmt.Errorf("failed to set PWM values: %w", err)
		}

		ch.on = on
		ch.off = off
		pca.logger.Detailed("SetPWM: канал %d успешно установлен", channel)
		return nil
	}
}

// SetAllPWM устанавливает одинаковые значения PWM для всех каналов.
func (pca *PCA9685) SetAllPWM(ctx context.Context, on, off uint16) error {
	pca.logger.Basic("SetAllPWM: установка всех каналов: on=%d, off=%d", on, off)
	pca.mu.Lock()
	defer pca.mu.Unlock()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		pca.logger.Error("SetAllPWM: контекст отменён: %v", err)
		return err
	default:
		data := []byte{
			byte(on & 0xFF),
			byte(on >> 8),
			byte(off & 0xFF),
			byte(off >> 8),
		}
		if err := pca.dev.WriteReg(RegAllLed, data); err != nil {
			pca.logger.Error("SetAllPWM: не удалось установить значения для всех каналов: %v", err)
			return fmt.Errorf("failed to set all PWM values: %w", err)
		}

		for i := range pca.channels {
			if pca.channels[i].enabled {
				pca.channels[i].on = on
				pca.channels[i].off = off
			}
		}
		pca.logger.Detailed("SetAllPWM: значения успешно установлены для всех каналов")
		return nil
	}
}

// SetMultiPWM устанавливает значения PWM для нескольких каналов.
func (pca *PCA9685) SetMultiPWM(ctx context.Context, settings map[int]struct{ On, Off uint16 }) error {
	pca.logger.Basic("SetMultiPWM: установка нескольких каналов")
	// Проверяем корректность номеров каналов.
	for channel := range settings {
		if err := pca.validateChannel(channel); err != nil {
			pca.logger.Error("SetMultiPWM: неверный номер канала %d: %v", channel, err)
			return err
		}
	}

	for channel, values := range settings {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			pca.logger.Error("SetMultiPWM: контекст отменён: %v", err)
			return err
		default:
			if err := pca.SetPWM(ctx, channel, values.On, values.Off); err != nil {
				pca.logger.Error("SetMultiPWM: не удалось установить PWM для канала %d: %v", channel, err)
				return fmt.Errorf("failed to set PWM for channel %d: %w", channel, err)
			}
		}
	}
	return nil
}

// EnableChannels включает указанные каналы.
func (pca *PCA9685) EnableChannels(channels ...int) error {
	pca.logger.Basic("Включение каналов: %v", channels)
	for _, ch := range channels {
		if err := pca.validateChannel(ch); err != nil {
			pca.logger.Error("EnableChannels: неверный номер канала %d: %v", ch, err)
			return err
		}
		pca.channels[ch].mu.Lock()
		pca.channels[ch].enabled = true
		pca.channels[ch].mu.Unlock()
	}
	return nil
}

// DisableChannels выключает указанные каналы.
func (pca *PCA9685) DisableChannels(channels ...int) error {
	pca.logger.Basic("Отключение каналов: %v", channels)
	for _, ch := range channels {
		if err := pca.validateChannel(ch); err != nil {
			pca.logger.Error("DisableChannels: неверный номер канала %d: %v", ch, err)
			return err
		}
		pca.channels[ch].mu.Lock()
		pca.channels[ch].enabled = false
		// При отключении устанавливаем нулевые значения PWM.
		if err := pca.SetPWM(pca.ctx, ch, 0, 0); err != nil {
			pca.channels[ch].mu.Unlock()
			pca.logger.Error("DisableChannels: не удалось отключить канал %d: %v", ch, err)
			return fmt.Errorf("failed to disable channel %d: %w", ch, err)
		}
		pca.channels[ch].mu.Unlock()
	}
	return nil
}

// GetChannelState возвращает состояние канала: включён ли, и текущие значения on/off.
func (pca *PCA9685) GetChannelState(channel int) (enabled bool, on, off uint16, err error) {
	pca.logger.Detailed("GetChannelState: получение состояния канала %d", channel)
	if err := pca.validateChannel(channel); err != nil {
		pca.logger.Error("GetChannelState: неверный номер канала %d: %v", channel, err)
		return false, 0, 0, err
	}

	ch := &pca.channels[channel]
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.enabled, ch.on, ch.off, nil
}

// validateChannel проверяет корректность номера канала (0–15).
func (pca *PCA9685) validateChannel(channel int) error {
	if channel < 0 || channel > 15 {
		return fmt.Errorf("invalid channel number: %d", channel)
	}
	return nil
}

// readMode1 считывает значение регистра MODE1.
func (pca *PCA9685) readMode1() (byte, error) {
	data := make([]byte, 1)
	if err := pca.dev.ReadReg(RegMode1, data); err != nil {
		pca.logger.Error("readMode1: не удалось прочитать MODE1: %v", err)
		return 0, fmt.Errorf("failed to read MODE1: %w", err)
	}
	pca.logger.Detailed("readMode1: получено значение 0x%X", data[0])
	return data[0], nil
}

// FadeChannel плавно изменяет значение PWM для указанного канала от start до end за duration.
func (pca *PCA9685) FadeChannel(ctx context.Context, channel int, start, end uint16, duration time.Duration) error {
	pca.logger.Basic("Начало плавного изменения (fade) на канале %d от %d до %d за %v", channel, start, end, duration)
	if err := pca.validateChannel(channel); err != nil {
		pca.logger.Error("FadeChannel: неверный номер канала %d: %v", channel, err)
		return err
	}
	steps := 20
	stepDuration := duration / time.Duration(steps)
	diff := int(end) - int(start)
	for i := 0; i <= steps; i++ {
		value := start + uint16(float64(diff)*float64(i)/float64(steps))
		if err := pca.SetPWM(ctx, channel, 0, value); err != nil {
			pca.logger.Error("FadeChannel: не удалось установить PWM на канале %d: %v", channel, err)
			return err
		}
		pca.logger.Detailed("FadeChannel: канал %d установлен на %d", channel, value)
		time.Sleep(stepDuration)
	}
	pca.logger.Basic("Завершено плавное изменение на канале %d", channel)
	return nil
}

// DumpState возвращает строку с текущим состоянием контроллера (частота и состояние каналов).
func (pca *PCA9685) DumpState() string {
	pca.mu.RLock()
	defer pca.mu.RUnlock()
	state := fmt.Sprintf("Состояние PCA9685: Частота: %f Гц\n", pca.Freq)
	for i := range pca.channels {
		ch := &pca.channels[i] // получаем указатель на элемент, чтобы не копировать мьютекс
		ch.mu.RLock()
		state += fmt.Sprintf("Канал %d: enabled=%v, on=%d, off=%d\n", i, ch.enabled, ch.on, ch.off)
		ch.mu.RUnlock()
	}
	pca.logger.Detailed("DumpState:\n%s", state)
	return state
}
