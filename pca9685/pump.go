package pca9685

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// Pump представляет управление насосом
type Pump struct {
	pca      *PCA9685
	channel  int
	minSpeed uint16
	maxSpeed uint16
	mu       sync.RWMutex
}

// NewPump создает новый контроллер насоса
func NewPump(pca *PCA9685, channel int, opts ...PumpOption) (*Pump, error) {
	if channel < 0 || channel > 15 {
		return nil, fmt.Errorf("invalid channel number: %d", channel)
	}

	pump := &Pump{
		pca:      pca,
		channel:  channel,
		minSpeed: 0,
		maxSpeed: 4095,
	}

	// Применение опций конфигурации
	for _, opt := range opts {
		opt(pump)
	}

	// Включение канала
	if err := pca.EnableChannels(channel); err != nil {
		return nil, fmt.Errorf("failed to enable channel: %w", err)
	}

	return pump, nil
}

// PumpOption определяет опцию конфигурации насоса
type PumpOption func(*Pump)

// WithSpeedLimits устанавливает минимальную и максимальную скорости насоса
func WithSpeedLimits(min, max uint16) PumpOption {
	return func(p *Pump) {
		if min > max {
			min, max = max, min
		}
		if max > 4095 {
			max = 4095
		}
		p.minSpeed = min
		p.maxSpeed = max
	}
}

// SetSpeed устанавливает скорость насоса (0-100%)
func (p *Pump) SetSpeed(ctx context.Context, percent float64) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("speed percentage must be between 0 and 100")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Более точное масштабирование
	scale := func(percent float64, min, max uint16) uint16 {
		range_ := float64(max - min)
		value := math.Round((percent * range_) / 100.0)
		return uint16(value) + min
	}

	value := scale(percent, p.minSpeed, p.maxSpeed)
	return p.pca.SetPWM(ctx, p.channel, 0, value)
}

// Stop останавливает насос
func (p *Pump) Stop(ctx context.Context) error {
	return p.SetSpeed(ctx, 0)
}

// GetCurrentSpeed возвращает текущую скорость насоса в процентах
func (p *Pump) GetCurrentSpeed() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, _, off, err := p.pca.GetChannelState(p.channel)
	if err != nil {
		return 0, fmt.Errorf("failed to get channel state: %w", err)
	}

	// Более точное обратное преобразование
	if off <= p.minSpeed {
		return 0, nil
	}
	if off >= p.maxSpeed {
		return 100, nil
	}

	range_ := float64(p.maxSpeed - p.minSpeed)
	percent := math.Round(float64(off-p.minSpeed) * 100.0 / range_)
	return percent, nil
}

// SetSpeedLimits устанавливает новые ограничения скорости
func (p *Pump) SetSpeedLimits(min, max uint16) error {
	if min > max {
		return fmt.Errorf("minimum speed cannot be greater than maximum speed")
	}
	if max > 4095 {
		return fmt.Errorf("maximum speed cannot exceed 4095")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.minSpeed = min
	p.maxSpeed = max
	return nil
}
