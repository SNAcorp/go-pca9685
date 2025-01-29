package pca9685

import (
	"context"
	"fmt"
	"image/color"
	"sync"
)

// RGBLed представляет RGB светодиод
type RGBLed struct {
	pca         *PCA9685
	channels    [3]int
	brightness  float64
	mu          sync.RWMutex
	calibration RGBCalibration
}

// RGBCalibration содержит калибровочные данные для RGB светодиода
type RGBCalibration struct {
	RedMin, RedMax     uint16
	GreenMin, GreenMax uint16
	BlueMin, BlueMax   uint16
}

// DefaultRGBCalibration возвращает калибровку по умолчанию
func DefaultRGBCalibration() RGBCalibration {
	return RGBCalibration{
		RedMax:   4095,
		GreenMax: 4095,
		BlueMax:  4095,
	}
}

// NewRGBLed создает новый RGB светодиод
func NewRGBLed(pca *PCA9685, red, green, blue int) (*RGBLed, error) {
	for _, ch := range []int{red, green, blue} {
		if ch < 0 || ch > 15 {
			return nil, fmt.Errorf("invalid channel number: %d", ch)
		}
	}

	led := &RGBLed{
		pca:         pca,
		channels:    [3]int{red, green, blue},
		brightness:  1.0,
		calibration: DefaultRGBCalibration(),
	}

	// Включение каналов
	if err := pca.EnableChannels(red, green, blue); err != nil {
		return nil, fmt.Errorf("failed to enable channels: %w", err)
	}

	return led, nil
}

// SetCalibration устанавливает калибровочные данные для светодиода
func (l *RGBLed) SetCalibration(cal RGBCalibration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calibration = cal
}

// GetCalibration возвращает текущие калибровочные данные
func (l *RGBLed) GetCalibration() RGBCalibration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.calibration
}

// SetColor устанавливает цвет в значениях RGB (0-255)
func (l *RGBLed) SetColor(ctx context.Context, r, g, b uint8) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Масштабирование значений с учетом калибровки и яркости
	scale := func(value uint8, min, max uint16) uint16 {
		v := float64(value) * l.brightness
		scaled := uint16((v * float64(max-min) / 255.0) + float64(min))
		if scaled > max {
			return max
		}
		return scaled
	}

	values := map[int]struct{ On, Off uint16 }{
		l.channels[0]: {0, scale(r, l.calibration.RedMin, l.calibration.RedMax)},
		l.channels[1]: {0, scale(g, l.calibration.GreenMin, l.calibration.GreenMax)},
		l.channels[2]: {0, scale(b, l.calibration.BlueMin, l.calibration.BlueMax)},
	}

	return l.pca.SetMultiPWM(ctx, values)
}

// SetColorStdlib устанавливает цвет используя стандартный пакет color
func (l *RGBLed) SetColorStdlib(ctx context.Context, c color.Color) error {
	r, g, b, _ := c.RGBA()
	return l.SetColor(ctx, uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// SetBrightness устанавливает яркость (0.0-1.0)
func (l *RGBLed) SetBrightness(brightness float64) error {
	if brightness < 0 || brightness > 1 {
		return fmt.Errorf("brightness must be between 0 and 1")
	}

	l.mu.Lock()
	l.brightness = brightness
	l.mu.Unlock()
	return nil
}

// GetBrightness возвращает текущую яркость
func (l *RGBLed) GetBrightness() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.brightness
}

// Off выключает все каналы светодиода
func (l *RGBLed) Off(ctx context.Context) error {
	return l.SetColor(ctx, 0, 0, 0)
}
