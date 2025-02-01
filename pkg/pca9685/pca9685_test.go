package pca9685

import (
	"context"
	"image/color"
	"math"
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
	pca, _ := New(adapter, DefaultConfig())

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
	pca, _ := New(adapter, DefaultConfig())
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
	pca, _ := New(adapter, DefaultConfig())
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
	pca, _ := New(adapter, DefaultConfig())
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
	pca, _ := New(adapter, DefaultConfig())
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
		pump, _ := NewPump(pca, 0)
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
		pump, _ := NewPump(pca, 0)
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
		pump, _ := NewPump(pca, 0)

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
	pca, _ := New(adapter, DefaultConfig())

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
	pca, _ := New(adapter, DefaultConfig())
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
