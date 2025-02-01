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
	if channel < 0 || channel > 15 {
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
		return nil, fmt.Errorf("failed to enable channel: %w", err)
	}

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
	}
}

// SetSpeed устанавливает скорость насоса в процентах (0–100%).
func (p *Pump) SetSpeed(ctx context.Context, percent float64) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("speed percentage must be between 0 and 100")
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
	return p.pca.SetPWM(ctx, p.channel, 0, value)
}

// Stop останавливает насос, устанавливая скорость 0%.
func (p *Pump) Stop(ctx context.Context) error {
	return p.SetSpeed(ctx, 0)
}

// GetCurrentSpeed возвращает текущую скорость насоса в процентах.
func (p *Pump) GetCurrentSpeed() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, _, off, err := p.pca.GetChannelState(p.channel)
	if err != nil {
		return 0, fmt.Errorf("failed to get channel state: %w", err)
	}

	// Обратное масштабирование.
	if off <= p.MinSpeed {
		return 0, nil
	}
	if off >= p.MaxSpeed {
		return 100, nil
	}

	range_ := float64(p.MaxSpeed - p.MinSpeed)
	percent := math.Round(float64(off-p.MinSpeed) * 100.0 / range_)
	return percent, nil
}

// SetSpeedLimits устанавливает новые ограничения скорости.
func (p *Pump) SetSpeedLimits(min, max uint16) error {
	if min > max {
		return fmt.Errorf("minimum speed cannot be greater than maximum speed")
	}
	if max > 4095 {
		return fmt.Errorf("maximum speed cannot exceed 4095")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.MinSpeed = min
	p.MaxSpeed = max
	return nil
}
