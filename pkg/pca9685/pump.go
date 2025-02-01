package pca9685

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// Pump представляет управление насосом.
type Pump struct {
	pca      *PCA9685
	channel  int
	MinSpeed uint16
	MaxSpeed uint16
	mu       sync.RWMutex
}

// NewPump создает новый контроллер насоса.
// При создании насоса проверяется корректность номера канала и опционально
// применяются опции (например, установка ограничений скорости).
func NewPump(pca *PCA9685, channel int, opts ...PumpOption) (*Pump, error) {
	pca.logger.Detailed("Создание нового насоса на канале: %d", channel)
	if channel < 0 || channel > 15 {
		pca.logger.Error("NewPump: неверный номер канала: %d", channel)
		return nil, fmt.Errorf("invalid channel number: %d", channel)
	}

	pump := &Pump{
		pca:      pca,
		channel:  channel,
		MinSpeed: 0,
		MaxSpeed: 4095,
	}

	// Применение опций конфигурации.
	for _, opt := range opts {
		opt(pump)
	}

	// Включение канала.
	if err := pca.EnableChannels(channel); err != nil {
		pca.logger.Error("NewPump: не удалось включить канал %d: %v", channel, err)
		return nil, fmt.Errorf("failed to enable channel: %w", err)
	}

	pca.logger.Basic("Насос успешно создан на канале: %d", channel)
	return pump, nil
}

// PumpOption определяет опцию конфигурации насоса.
type PumpOption func(*Pump)

// WithSpeedLimits устанавливает минимальную и максимальную скорости насоса.
func WithSpeedLimits(min, max uint16) PumpOption {
	return func(p *Pump) {
		if min > max {
			min, max = max, min
		}
		if max > 4095 {
			max = 4095
		}
		p.MinSpeed = min
		p.MaxSpeed = max
		p.pca.logger.Detailed("WithSpeedLimits: установлены ограничения скорости: min=%d, max=%d", min, max)
	}
}

// SetSpeed устанавливает скорость насоса в процентах (0–100%).
func (p *Pump) SetSpeed(ctx context.Context, percent float64) error {
	p.pca.logger.Detailed("SetSpeed: установка скорости насоса на %f%%", percent)
	if percent < 0 || percent > 100 {
		err := fmt.Errorf("speed percentage must be between 0 and 100")
		p.pca.logger.Error("SetSpeed: неверное значение скорости: %f%%", percent)
		return err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Масштабирование: вычисляем значение PWM на основе процентов.
	scale := func(percent float64, min, max uint16) uint16 {
		range_ := float64(max - min)
		value := math.Round((percent * range_) / 100.0)
		return uint16(value) + min
	}

	value := scale(percent, p.MinSpeed, p.MaxSpeed)
	p.pca.logger.Detailed("SetSpeed: вычисленное значение PWM: %d", value)
	if err := p.pca.SetPWM(ctx, p.channel, 0, value); err != nil {
		p.pca.logger.Error("SetSpeed: ошибка установки PWM: %v", err)
		return err
	}
	p.pca.logger.Basic("SetSpeed: скорость насоса установлена на %f%%", percent)
	return nil
}

// Stop останавливает насос, устанавливая скорость 0%.
func (p *Pump) Stop(ctx context.Context) error {
	p.pca.logger.Basic("Stop: остановка насоса на канале %d", p.channel)
	if err := p.SetSpeed(ctx, 0); err != nil {
		p.pca.logger.Error("Stop: ошибка остановки насоса: %v", err)
		return err
	}
	return nil
}

// GetCurrentSpeed возвращает текущую скорость насоса в процентах.
func (p *Pump) GetCurrentSpeed() (float64, error) {
	p.pca.logger.Detailed("GetCurrentSpeed: получение текущей скорости насоса на канале %d", p.channel)
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, _, off, err := p.pca.GetChannelState(p.channel)
	if err != nil {
		p.pca.logger.Error("GetCurrentSpeed: ошибка получения состояния канала %d: %v", p.channel, err)
		return 0, fmt.Errorf("failed to get channel state: %w", err)
	}

	// Обратное масштабирование.
	var percent float64
	if off <= p.MinSpeed {
		percent = 0
	} else if off >= p.MaxSpeed {
		percent = 100
	} else {
		range_ := float64(p.MaxSpeed - p.MinSpeed)
		percent = math.Round(float64(off-p.MinSpeed) * 100.0 / range_)
	}
	p.pca.logger.Detailed("GetCurrentSpeed: получена скорость %f%% для канала %d", percent, p.channel)
	return percent, nil
}

// SetSpeedLimits устанавливает новые ограничения скорости.
func (p *Pump) SetSpeedLimits(min, max uint16) error {
	p.pca.logger.Detailed("SetSpeedLimits: установка новых ограничений скорости: min=%d, max=%d", min, max)
	if min > max {
		err := fmt.Errorf("minimum speed cannot be greater than maximum speed")
		p.pca.logger.Error("SetSpeedLimits: ошибка установки ограничений: min (%d) больше max (%d)", min, max)
		return err
	}
	if max > 4095 {
		err := fmt.Errorf("maximum speed cannot exceed 4095")
		p.pca.logger.Error("SetSpeedLimits: ошибка установки ограничений: max (%d) превышает 4095", max)
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.MinSpeed = min
	p.MaxSpeed = max
	p.pca.logger.Basic("SetSpeedLimits: ограничения скорости успешно установлены: min=%d, max=%d", min, max)
	return nil
}
