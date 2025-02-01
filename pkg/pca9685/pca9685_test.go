package pca9685

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPCA9685_New(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	config := DefaultConfig()

	pca, err := New(adapter, config)
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}
	if pca == nil {
		t.Fatal("Expected non-nil PCA9685")
	}

	if pca.Freq != config.InitialFreq {
		t.Errorf("Expected frequency %v, got %v", config.InitialFreq, pca.Freq)
	}
}

func TestPCA9685_SetPWMFreq(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	tests := []struct {
		name    string
		freq    float64
		wantErr bool
	}{
		{"Valid frequency", 1000, false},
		{"Minimum frequency", 24, false},
		{"Maximum frequency", 1526, false},
		{"Below minimum", 23, true},
		{"Above maximum", 1527, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pca.SetPWMFreq(tt.freq)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPWMFreq() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && pca.Freq != tt.freq {
				t.Errorf("SetPWMFreq() freq = %v, want %v", pca.Freq, tt.freq)
			}
		})
	}
}

func TestPCA9685_SetPWM(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		channel int
		on      uint16
		off     uint16
		wantErr bool
	}{
		{"Valid channel", 0, 0, 2048, false},
		{"Invalid channel low", -1, 0, 0, true},
		{"Invalid channel high", 16, 0, 0, true},
		{"Full range", 1, 0, 4095, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pca.SetPWM(ctx, tt.channel, tt.on, tt.off)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPWM() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPCA9685_SetMultiPWM(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx := context.Background()

	settings := map[int]struct{ On, Off uint16 }{
		0: {0, 2048},
		1: {0, 4095},
		2: {2048, 4095},
	}

	if err := pca.SetMultiPWM(ctx, settings); err != nil {
		t.Errorf("SetMultiPWM() error = %v", err)
	}

	invalidSettings := map[int]struct{ On, Off uint16 }{
		-1: {0, 2048},
		16: {0, 4095},
	}

	if err := pca.SetMultiPWM(ctx, invalidSettings); err == nil {
		t.Error("SetMultiPWM() expected error for invalid channels")
	}
}

func TestRGBLed(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx := context.Background()

	led, err := NewRGBLed(pca, 0, 1, 2)
	if err != nil {
		t.Fatalf("NewRGBLed() error = %v", err)
	}

	tests := []struct {
		name    string
		r, g, b uint8
		wantErr bool
	}{
		{"Black", 0, 0, 0, false},
		{"White", 255, 255, 255, false},
		{"Red", 255, 0, 0, false},
		{"Green", 0, 255, 0, false},
		{"Blue", 0, 0, 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := led.SetColor(ctx, tt.r, tt.g, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetColor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("SetColorStdlib", func(t *testing.T) {
		c := color.RGBA{R: 128, G: 64, B: 32, A: 255}
		if err := led.SetColorStdlib(ctx, c); err != nil {
			t.Errorf("SetColorStdlib() error = %v", err)
		}
	})

	t.Run("Brightness", func(t *testing.T) {
		if err := led.SetBrightness(0.5); err != nil {
			t.Errorf("SetBrightness() error = %v", err)
		}
		if b := led.GetBrightness(); b != 0.5 {
			t.Errorf("GetBrightness() = %v, want %v", b, 0.5)
		}
		if err := led.SetBrightness(-0.1); err == nil {
			t.Error("SetBrightness() expected error for negative value")
		}
		if err := led.SetBrightness(1.1); err == nil {
			t.Error("SetBrightness() expected error for value > 1")
		}
	})
}

func TestPump(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx := context.Background()

	t.Run("Creation", func(t *testing.T) {
		pump, err := NewPump(pca, 0, WithSpeedLimits(1000, 3000))
		if err != nil {
			t.Fatalf("NewPump() error = %v", err)
		}
		if pump.MinSpeed != 1000 || pump.MaxSpeed != 3000 {
			t.Errorf("Speed limits not set correctly, got min %v, max %v", pump.MinSpeed, pump.MaxSpeed)
		}
	})

	t.Run("SetSpeed", func(t *testing.T) {
		pump, err := NewPump(pca, 0)

		if err != nil {
			t.Fatalf("NewPump() error = %v", err)
		}

		tests := []struct {
			name    string
			speed   float64
			wantErr bool
		}{
			{"Zero speed", 0, false},
			{"Half speed", 50, false},
			{"Full speed", 100, false},
			{"Invalid negative", -1, true},
			{"Invalid over 100", 101, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := pump.SetSpeed(ctx, tt.speed)
				if (err != nil) != tt.wantErr {
					t.Errorf("SetSpeed() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("GetCurrentSpeed", func(t *testing.T) {
		pump, err := NewPump(pca, 0)

		if err != nil {
			t.Fatalf("NewPump() error = %v", err)
		}
		targetSpeed := 50.0

		if err := pump.SetSpeed(ctx, targetSpeed); err != nil {
			t.Fatalf("SetSpeed() error = %v", err)
		}

		currentSpeed, err := pump.GetCurrentSpeed()
		if err != nil {
			t.Errorf("GetCurrentSpeed() error = %v", err)
		}

		const epsilon = 0.01
		if diff := math.Abs(currentSpeed - targetSpeed); diff > epsilon {
			t.Errorf("GetCurrentSpeed() = %.2f, want %.2f (diff: %.2f)", currentSpeed, targetSpeed, diff)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		pump, err := NewPump(pca, 0)

		if err != nil {
			t.Fatalf("NewPump() error = %v", err)
		}

		if err := pump.SetSpeed(ctx, 50); err != nil {
			t.Fatalf("SetSpeed() error = %v", err)
		}

		if err := pump.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}

		if speed, err := pump.GetCurrentSpeed(); err != nil || speed != 0 {
			t.Errorf("After Stop(): speed = %v, error = %v", speed, err)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pca.SetPWM(ctx, 0, 0, 2048); err == nil {
		t.Error("SetPWM() should fail with cancelled context")
	}

	if err := pca.SetAllPWM(ctx, 0, 2048); err == nil {
		t.Error("SetAllPWM() should fail with cancelled context")
	}

	settings := map[int]struct{ On, Off uint16 }{
		0: {0, 2048},
	}
	if err := pca.SetMultiPWM(ctx, settings); err == nil {
		t.Error("SetMultiPWM() should fail with cancelled context")
	}
}

func TestConcurrency(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing")
	pca, err := New(adapter, DefaultConfig())

	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx := context.Background()

	const numGoroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				channel := n % 16
				if err := pca.SetPWM(ctx, channel, 0, uint16(j)); err != nil {
					t.Errorf("SetPWM() error in goroutine %d: %v", n, err)
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}

func TestPCA9685_FadeChannel(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing FadeChannel")
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}
	ctx := context.Background()
	channel := 0
	start := uint16(0)
	end := uint16(3000)
	duration := 100 * time.Millisecond

	// Set the initial PWM value
	if err := pca.SetPWM(ctx, channel, 0, start); err != nil {
		t.Fatalf("SetPWM failed: %v", err)
	}

	// Invoke FadeChannel to gradually change PWM from start to end
	if err := pca.FadeChannel(ctx, channel, start, end, duration); err != nil {
		t.Fatalf("FadeChannel failed: %v", err)
	}

	// Check that the PWM value at the channel is now equal to 'end'
	_, _, off, err := pca.GetChannelState(channel)
	if err != nil {
		t.Fatalf("GetChannelState failed: %v", err)
	}
	if off != end {
		t.Errorf("FadeChannel: expected off=%d, got %d", end, off)
	}
}

func TestPCA9685_FadeChannel_Cancel(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing FadeChannel with cancelled context")
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}
	// Create a context that will be cancelled shortly
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context after a short delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err = pca.FadeChannel(ctx, 0, 0, 3000, 100*time.Millisecond)
	if err == nil {
		t.Error("Expected FadeChannel to return an error due to cancelled context")
	} else {
		t.Logf("FadeChannel correctly returned error on cancelled context: %v", err)
	}
}

func TestPCA9685_DumpState(t *testing.T) {
	adapter := NewTestI2C()
	t.Log("Using TestI2C adapter for testing DumpState")
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}
	ctx := context.Background()

	// Set distinct PWM values on several channels
	for i := 0; i < 8; i++ {
		if err := pca.SetPWM(ctx, i, 0, uint16(i*250)); err != nil {
			t.Errorf("SetPWM failed for channel %d: %v", i, err)
		}
	}

	state := pca.DumpState()
	if state == "" {
		t.Error("DumpState returned an empty string")
	}
	if !strings.Contains(state, "Состояние PCA9685:") {
		t.Error("DumpState output missing header 'Состояние PCA9685:'")
	}
	t.Logf("DumpState output:\n%s", state)
}

// DummyI2CDevice simulates an I2C device for testing I2CAdapterD2r2 and I2CAdapterD2r2Extended.
type DummyI2CDevice struct {
	mu          sync.Mutex
	writtenData []byte
	readData    []byte
	writeFail   int // number of times to fail WriteBytes
	readFail    int // number of times to fail ReadBytes
}

func (d *DummyI2CDevice) WriteBytes(data []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.writeFail > 0 {
		d.writeFail--
		return 0, errors.New("simulated write error")
	}
	d.writtenData = append(d.writtenData, data...)
	return len(data), nil
}

func (d *DummyI2CDevice) ReadBytes(data []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.readFail > 0 {
		d.readFail--
		return 0, errors.New("simulated read error")
	}
	n := copy(data, d.readData)
	return n, nil
}

func (d *DummyI2CDevice) Close() error {
	return nil
}

// DummyPeriphI2CDev simulates a periph.io I2C device.
type DummyPeriphI2CDev struct {
	mu          sync.Mutex
	lastWritten []byte
	txData      []byte
	txFail      int // number of times to fail Tx
}

func (d *DummyPeriphI2CDev) Tx(w, r []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.txFail > 0 {
		d.txFail--
		return errors.New("simulated Tx error")
	}
	if r == nil {
		d.lastWritten = append([]byte{}, w...)
		return nil
	}
	_ = copy(r, d.txData)
	return nil
}

// ===== Tests for TestI2C Adapter =====

func TestTestI2C_WriteRead(t *testing.T) {
	adapter := NewTestI2C()
	reg := uint8(0x05)
	writeData := []byte{1, 2, 3}
	if err := adapter.WriteReg(reg, writeData); err != nil {
		t.Fatalf("TestI2C WriteReg failed: %v", err)
	}
	readBuf := make([]byte, len(writeData))
	if err := adapter.ReadReg(reg, readBuf); err != nil {
		t.Fatalf("TestI2C ReadReg failed: %v", err)
	}
	if string(readBuf) != string(writeData) {
		t.Errorf("TestI2C ReadReg: expected %v, got %v", writeData, readBuf)
	}
}

func TestTestI2C_ReadNotFound(t *testing.T) {
	adapter := NewTestI2C()
	reg := uint8(0x10)
	readBuf := make([]byte, 5)
	if err := adapter.ReadReg(reg, readBuf); err != nil {
		t.Fatalf("TestI2C ReadReg failed: %v", err)
	}
	expected := []byte{0, 0, 0, 0, 0}
	if string(readBuf) != string(expected) {
		t.Errorf("TestI2C ReadReg: expected %v, got %v", expected, readBuf)
	}
}

func TestTestI2C_Close(t *testing.T) {
	adapter := NewTestI2C()
	if err := adapter.Close(); err != nil {
		t.Errorf("TestI2C Close failed: %v", err)
	}
}

func TestConcurrentSameChannelAccess(t *testing.T) {
	adapter := NewTestI2C()
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	const (
		numGoroutines = 10
		numIterations = 100
		channel       = 0
	)

	// Создаем мьютекс для синхронизации доступа к каналу PWM
	var pwmMutex sync.Mutex

	// Канал для сбора результатов
	results := make(chan struct {
		routineID int
		values    []uint16
	}, numGoroutines)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Запускаем горутины
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			localValues := make([]uint16, 0, numIterations)

			baseValue := uint16(routineID * numIterations)
			for j := 0; j < numIterations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					value := baseValue + uint16(j)

					// Защищаем доступ к PWM мьютексом
					pwmMutex.Lock()
					err := pca.SetPWM(ctx, channel, 0, value)
					if err != nil {
						t.Errorf("Goroutine %d failed to set PWM: %v", routineID, err)
						pwmMutex.Unlock()
						return
					}

					// Проверяем состояние
					enabled, _, _, err := pca.GetChannelState(channel)
					if err != nil {
						t.Errorf("Goroutine %d failed to get channel state: %v", routineID, err)
						pwmMutex.Unlock()
						return
					}
					pwmMutex.Unlock()

					if !enabled {
						t.Errorf("Channel unexpectedly disabled in goroutine %d", routineID)
						return
					}

					// Сохраняем значение
					localValues = append(localValues, value)
				}
			}

			// Отправляем результаты
			results <- struct {
				routineID int
				values    []uint16
			}{routineID, localValues}
		}(i)
	}

	// Закрываем канал результатов после завершения всех горутин
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем и проверяем результаты
	for result := range results {
		if len(result.values) != numIterations {
			t.Errorf("Goroutine %d: expected %d values, got %d",
				result.routineID, numIterations, len(result.values))
			continue
		}

		baseValue := uint16(result.routineID * numIterations)
		for idx, value := range result.values {
			expectedValue := baseValue + uint16(idx)
			if value != expectedValue {
				t.Errorf("Goroutine %d: at index %d expected value %d, got %d",
					result.routineID, idx, expectedValue, value)
			}
		}
	}
}

// TestMultipleChannelsConcurrent проверяет конкурентный доступ к разным каналам
func TestMultipleChannelsConcurrent(t *testing.T) {
	adapter := NewTestI2C()
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	const (
		numChannels             = 16
		numIterationsPerChannel = 50
	)

	var wg sync.WaitGroup
	errorsList := make(chan error, numChannels)
	ctx := context.Background()

	// Запускаем горутину для каждого канала
	for ch := 0; ch < numChannels; ch++ {
		wg.Add(1)
		go func(channel int) {
			defer wg.Done()

			// Массив для хранения установленных значений
			values := make([]uint16, 0, numIterationsPerChannel)

			for i := 0; i < numIterationsPerChannel; i++ {
				// Уникальное значение для этого канала и итерации
				value := uint16((channel * 256) + i)

				// Устанавливаем значение
				if err := pca.SetPWM(ctx, channel, 0, value); err != nil {
					errorsList <- fmt.Errorf("channel %d set error: %v", channel, err)
					return
				}

				// Проверяем значение
				_, _, off, err := pca.GetChannelState(channel)
				if err != nil {
					errorsList <- fmt.Errorf("channel %d get error: %v", channel, err)
					return
				}

				if off != value {
					errorsList <- fmt.Errorf("channel %d value mismatch: got %d, want %d",
						channel, off, value)
					return
				}

				values = append(values, off)
				time.Sleep(time.Millisecond)
			}

			// Проверяем финальное состояние канала
			enabled, _, finalValue, err := pca.GetChannelState(channel)
			if err != nil {
				errorsList <- fmt.Errorf("channel %d final state error: %v", channel, err)
				return
			}

			if !enabled {
				errorsList <- fmt.Errorf("channel %d unexpectedly disabled", channel)
				return
			}

			expectedFinal := uint16((channel * 256) + numIterationsPerChannel - 1)
			if finalValue != expectedFinal {
				errorsList <- fmt.Errorf("channel %d final value mismatch: got %d, want %d",
					channel, finalValue, expectedFinal)
				return
			}
		}(ch)
	}

	// Ждем завершения всех горутин
	wg.Wait()
	close(errorsList)

	// Проверяем наличие ошибок
	var errorList []error
	for err := range errorsList {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Found %d errors:", len(errorList))
		for i, err := range errorList {
			t.Errorf("Error %d: %v", i+1, err)
		}
	}
}

// TestConcurrentFrequencyChange проверяет устойчивость к изменениям частоты
func TestConcurrentFrequencyChange(t *testing.T) {
	adapter := NewTestI2C()
	pca, err := New(adapter, DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create PCA9685: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errorsList := make(chan error, 3)

	// Горутина изменения частоты
	wg.Add(1)
	go func() {
		defer wg.Done()
		frequencies := []float64{200, 500, 1000, 1500}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, freq := range frequencies {
					if err := pca.SetPWMFreq(freq); err != nil {
						errorsList <- fmt.Errorf("frequency error: %v", err)
						return
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()

	// Горутина записи PWM
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for ch := 0; ch < 16; ch++ {
					value := uint16(ch * 256)
					if err := pca.SetPWM(ctx, ch, 0, value); err != nil {
						errorsList <- fmt.Errorf("PWM error on channel %d: %v", ch, err)
						return
					}
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	// Горутина чтения состояния
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for ch := 0; ch < 16; ch++ {
					if _, _, _, err := pca.GetChannelState(ch); err != nil {
						errorsList <- fmt.Errorf("state error on channel %d: %v", ch, err)
						return
					}
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	// Ожидаем завершение контекста
	<-ctx.Done()
	wg.Wait()
	close(errorsList)

	// Проверяем наличие ошибок
	var errorList []error
	for err := range errorsList {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Found %d errors during concurrent operations:", len(errorList))
		for i, err := range errorList {
			t.Errorf("Error %d: %v", i+1, err)
		}
	}
}
