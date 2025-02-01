package pca9685

import (
	"context"
	"fmt"
	"image/color"
	"sync"
)

// RGBLed представляет RGB светодиод, управляемый через контроллер PCA9685.
type RGBLed struct {
	pca         *PCA9685
	channels    [3]int
	brightness  float64
	mu          sync.RWMutex
	calibration RGBCalibration
}

// RGBCalibration содержит калибровочные данные для RGB светодиода.
type RGBCalibration struct {
	RedMin, RedMax     uint16
	GreenMin, GreenMax uint16
	BlueMin, BlueMax   uint16
}

// DefaultRGBCalibration возвращает калибровку по умолчанию.
func DefaultRGBCalibration() RGBCalibration {
	return RGBCalibration{
		RedMax:   4095,
		GreenMax: 4095,
		BlueMax:  4095,
	}
}

// NewRGBLed создает новый RGB светодиод на указанных каналах (от 0 до 15).
func NewRGBLed(pca *PCA9685, red, green, blue int) (*RGBLed, error) {
	pca.logger.Detailed("Создание нового RGBLed на каналах: %d, %d, %d", red, green, blue)
	for _, ch := range []int{red, green, blue} {
		if ch < 0 || ch > 15 {
			pca.logger.Error("NewRGBLed: неверный номер канала: %d", ch)
			return nil, fmt.Errorf("invalid channel number: %d", ch)
		}
	}

	led := &RGBLed{
		pca:         pca,
		channels:    [3]int{red, green, blue},
		brightness:  1.0,
		calibration: DefaultRGBCalibration(),
	}

	// Включение каналов.
	if err := pca.EnableChannels(red, green, blue); err != nil {
		pca.logger.Error("NewRGBLed: не удалось включить каналы: %v", err)
		return nil, fmt.Errorf("failed to enable channels: %w", err)
	}

	pca.logger.Basic("RGBLed успешно создан на каналах: %d, %d, %d", red, green, blue)
	return led, nil
}

// SetCalibration устанавливает калибровочные данные для светодиода.
func (l *RGBLed) SetCalibration(cal RGBCalibration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pca.logger.Detailed("Установка калибровки для RGBLed: %+v", cal)
	l.calibration = cal
}

// GetCalibration возвращает текущие калибровочные данные.
func (l *RGBLed) GetCalibration() RGBCalibration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	cal := l.calibration
	l.pca.logger.Detailed("Получена калибровка для RGBLed: %+v", cal)
	return cal
}

// SetColor устанавливает цвет светодиода (значения RGB от 0 до 255).
func (l *RGBLed) SetColor(ctx context.Context, r, g, b uint8) error {
	l.pca.logger.Detailed("SetColor: установка цвета R=%d, G=%d, B=%d", r, g, b)
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Масштабирование с учетом калибровки и яркости.
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

	if err := l.pca.SetMultiPWM(ctx, values); err != nil {
		l.pca.logger.Error("SetColor: ошибка установки цвета: %v", err)
		return err
	}
	l.pca.logger.Detailed("SetColor: цвет успешно установлен")
	return nil
}

// SetColorStdlib устанавливает цвет с использованием стандартного пакета color.
func (l *RGBLed) SetColorStdlib(ctx context.Context, c color.Color) error {
	l.pca.logger.Detailed("SetColorStdlib: установка цвета через стандартный пакет color")
	r, g, b, _ := c.RGBA()
	// Приведение к 8-битному значению.
	if err := l.SetColor(ctx, uint8(r>>8), uint8(g>>8), uint8(b>>8)); err != nil {
		l.pca.logger.Error("SetColorStdlib: ошибка установки цвета: %v", err)
		return err
	}
	return nil
}

// SetBrightness устанавливает яркость (от 0.0 до 1.0).
func (l *RGBLed) SetBrightness(brightness float64) error {
	l.pca.logger.Detailed("SetBrightness: установка яркости: %f", brightness)
	if brightness < 0 || brightness > 1 {
		err := fmt.Errorf("brightness must be between 0 and 1")
		l.pca.logger.Error("SetBrightness: ошибка установки яркости: %v", err)
		return err
	}

	l.mu.Lock()
	l.brightness = brightness
	l.mu.Unlock()
	l.pca.logger.Detailed("SetBrightness: яркость успешно установлена")
	return nil
}

// GetBrightness возвращает текущую яркость.
func (l *RGBLed) GetBrightness() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	brightness := l.brightness
	l.pca.logger.Detailed("GetBrightness: текущая яркость: %f", brightness)
	return brightness
}

// Off выключает все каналы светодиода.
func (l *RGBLed) Off(ctx context.Context) error {
	l.pca.logger.Basic("Off: выключение RGBLed")
	if err := l.SetColor(ctx, 0, 0, 0); err != nil {
		l.pca.logger.Error("Off: ошибка выключения RGBLed: %v", err)
		return err
	}
	return nil
}

// On включает все каналы светодиода.
func (l *RGBLed) On(ctx context.Context) error {
	l.pca.logger.Basic("On: включение RGBLed")
	if err := l.SetColor(ctx, 255, 255, 255); err != nil {
		l.pca.logger.Error("On: ошибка включения RGBLed: %v", err)
		return err
	}
	return nil
}
