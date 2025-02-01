# Примеры использования библиотеки PCA9685

## Основное использование

### Инициализация устройства

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/snaart/pca9685"
)

func main() {
    // Создание базового контекста с таймаутом
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Создание I2C устройства (зависит от вашей реализации)
    i2c := NewI2CDevice()

    // Настройка конфигурации с расширенными опциями
    config := pca9685.DefaultConfig()
    config.InitialFreq = 1000    // Установка частоты PWM
    config.InvertLogic = false   // Прямая логика
    config.OpenDrain = false     // Использовать push-pull выходы
    config.Context = ctx         // Контекст для отмены операций
    
    // Создание контроллера
    pca, err := pca9685.New(i2c, config)
    if err != nil {
        log.Fatalf("Failed to initialize PCA9685: %v", err)
    }
    defer pca.Close()
}
```

### Управление RGB светодиодом

```go
// Создание RGB светодиода (каналы 0, 1, 2)
led, err := pca9685.NewRGBLed(pca, 0, 1, 2)
if err != nil {
    log.Fatalf("Failed to create RGB LED: %v", err)
}

// Калибровка светодиода для точной цветопередачи
cal := pca9685.RGBCalibration{
    RedMin: 0, RedMax: 4095,
    GreenMin: 0, GreenMax: 3800, // Компенсация если зеленый ярче
    BlueMin: 100, BlueMax: 4095, // Компенсация мертвой зоны синего
}
led.SetCalibration(cal)

// Создание контекста с таймаутом для операций
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

// Установка цвета (R, G, B: 0-255)
if err := led.SetColor(ctx, 255, 0, 0); err != nil { // Красный цвет
    switch {
    case err == context.DeadlineExceeded:
        log.Printf("Operation timed out")
    case err == context.Canceled:
        log.Printf("Operation was canceled")
    default:
        log.Printf("Failed to set color: %v", err)
    }
}

// Использование стандартного пакета color с поддержкой alpha
c := color.RGBA{R: 0, G: 255, B: 0, A: 255} // Зеленый цвет
if err := led.SetColorStdlib(ctx, c); err != nil {
    log.Printf("Failed to set color: %v", err)
}

// Управление яркостью с проверкой текущего значения
if err := led.SetBrightness(0.5); err != nil { // 50% яркости
    log.Printf("Failed to set brightness: %v", err)
}
currentBrightness := led.GetBrightness()
log.Printf("Current brightness: %.1f%%", currentBrightness*100)

// Выключение светодиода
if err := led.Off(ctx); err != nil {
    log.Printf("Failed to turn off LED: %v", err)
}
```

### Управление насосом с улучшенной точностью

```go
// Создание насоса с ограничениями скорости и плавным стартом
pump, err := pca9685.NewPump(pca, 3, 
    pca9685.WithSpeedLimits(500, 3500)) // Минимальная и максимальная скорости
if err != nil {
    log.Fatalf("Failed to create pump: %v", err)
}

ctx := context.Background()

// Установка скорости (0-100%) с высокой точностью
targetSpeed := 75.5 // 75.5%
if err := pump.SetSpeed(ctx, targetSpeed); err != nil {
    log.Printf("Failed to set pump speed: %v", err)
}

// Получение текущей скорости с проверкой точности
speed, err := pump.GetCurrentSpeed()
if err != nil {
    log.Printf("Failed to get pump speed: %v", err)
} else {
    // Проверка допустимой погрешности
    const epsilon = 0.01
    if diff := math.Abs(speed - targetSpeed); diff > epsilon {
        log.Printf("Warning: speed deviation %.2f%% exceeds tolerance", diff)
    }
    log.Printf("Current pump speed: %.2f%%", speed)
}

// Плавный запуск насоса
if err := SoftStart(ctx, pump, targetSpeed); err != nil {
    log.Printf("Soft start failed: %v", err)
}

// Остановка насоса
if err := pump.Stop(ctx); err != nil {
    log.Printf("Failed to stop pump: %v", err)
}

// Изменение ограничений скорости во время работы
if err := pump.SetSpeedLimits(1000, 3000); err != nil {
    log.Printf("Failed to update speed limits: %v", err)
}
```

### Одновременное управление несколькими каналами

```go
// Установка нескольких каналов одновременно с timeout
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

settings := map[int]struct{ On, Off uint16 }{
    0: {0, 2048},  // 50% на канале 0
    1: {0, 4095},  // 100% на канале 1
    2: {2048, 4095}, // Особый паттерн на канале 2
}

if err := pca.SetMultiPWM(ctx, settings); err != nil {
    if err == context.DeadlineExceeded {
        log.Printf("Operation timed out")
    } else {
        log.Printf("Failed to set multiple channels: %v", err)
    }
}

// Установка всех каналов в одно значение
if err := pca.SetAllPWM(ctx, 0, 2048); err != nil {
    log.Printf("Failed to set all channels: %v", err)
}

// Выборочное включение/выключение каналов с проверкой состояния
if err := pca.EnableChannels(0, 1, 2); err != nil {
    log.Printf("Failed to enable channels: %v", err)
}

// Проверка состояния канала
if enabled, on, off, err := pca.GetChannelState(0); err != nil {
    log.Printf("Failed to get channel state: %v", err)
} else {
    log.Printf("Channel 0: enabled=%v, on=%d, off=%d", enabled, on, off)
}

if err := pca.DisableChannels(3, 4); err != nil {
    log.Printf("Failed to disable channels: %v", err)
}
```

## Расширенные примеры

### Анимация RGB светодиода с плавными переходами

```go
func RainbowEffect(ctx context.Context, led *pca9685.RGBLed) error {
    // HSV -> RGB конвертер с улучшенной точностью
    hsvToRgb := func(h, s, v float64) (uint8, uint8, uint8) {
        // ... (реализация конвертера)
    }

    // Бесконечный цикл анимации с поддержкой отмены
    h := 0.0
    ticker := time.NewTicker(20 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            r, g, b := hsvToRgb(h, 1.0, 1.0)
            if err := led.SetColor(ctx, r, g, b); err != nil {
                return fmt.Errorf("failed to set color: %w", err)
            }
            h += 0.01
            if h >= 1.0 {
                h = 0
            }
        }
    }
}
```

### Плавный запуск насоса с контролем ускорения

```go
func SoftStart(ctx context.Context, pump *pca9685.Pump, targetSpeed float64) error {
    const (
        startSpeed = 20.0  // Начальная скорость
        step      = 1.0    // Шаг увеличения
        delay     = 100 * time.Millisecond
    )

    // Проверка текущей скорости
    currentSpeed, err := pump.GetCurrentSpeed()
    if err != nil {
        return fmt.Errorf("failed to get current speed: %w", err)
    }

    // Если текущая скорость выше начальной, начинаем с неё
    startFrom := math.Max(currentSpeed, startSpeed)
    
    ticker := time.NewTicker(delay)
    defer ticker.Stop()

    for speed := startFrom; speed <= targetSpeed; speed += step {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := pump.SetSpeed(ctx, speed); err != nil {
                return fmt.Errorf("failed to set speed: %w", err)
            }
            
            // Проверка реальной скорости
            actual, err := pump.GetCurrentSpeed()
            if err != nil {
                return fmt.Errorf("failed to verify speed: %w", err)
            }
            
            // Проверка отклонения
            if math.Abs(actual-speed) > 2.0 {
                return fmt.Errorf("speed deviation too high")
            }
        }
    }
    return nil
}
```

## Обработка ошибок и мониторинг

```go
// Расширенный обработчик ошибок с поддержкой всех типов ошибок
func handleDeviceErrors(err error) {
    switch {
    case err == nil:
        return
    case errors.Is(err, context.Canceled):
        log.Println("Operation was canceled by user")
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Operation timed out")
    default:
        var deviceErr *pca9685.DeviceError
        if errors.As(err, &deviceErr) {
            log.Printf("Device error: %v (code: %d)", deviceErr, deviceErr.Code)
        } else {
            log.Printf("Unknown error: %v", err)
        }
    }
}

// Мониторинг состояния устройства
type DeviceMonitor struct {
    pca  *pca9685.PCA9685
    done chan struct{}
}

func NewDeviceMonitor(pca *pca9685.PCA9685) *DeviceMonitor {
    return &DeviceMonitor{
        pca:  pca,
        done: make(chan struct{}),
    }
}

func (m *DeviceMonitor) Start(ctx context.Context) {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Проверка состояния каналов
            for ch := 0; ch < 16; ch++ {
                if enabled, _, _, err := m.pca.GetChannelState(ch); err != nil {
                    log.Printf("Failed to check channel %d: %v", ch, err)
                } else if enabled {
                    // Мониторинг активных каналов
                    log.Printf("Channel %d is active", ch)
                }
            }
        }
    }
}
```

## Использование с периферийными устройствами

```go
// Пример управления системой охлаждения
type CoolingSystem struct {
    led      *pca9685.RGBLed    // Индикатор состояния
    pump     *pca9685.Pump      // Насос охлаждения
    mu       sync.RWMutex       // Защита состояния
    status   bool               // Текущий статус
}

func NewCoolingSystem(pca *pca9685.PCA9685) (*CoolingSystem, error) {
    led, err := pca9685.NewRGBLed(pca, 0, 1, 2)
    if err != nil {
        return nil, fmt.Errorf("failed to create LED: %w", err)
    }

    pump, err := pca9685.NewPump(pca, 3, 
        pca9685.WithSpeedLimits(1000, 3500))
    if err != nil {
        return nil, fmt.Errorf("failed to create pump: %w", err)
    }

    return &CoolingSystem{
        led:    led,
        pump:   pump,
        status: false,
    }, nil
}

func (cs *CoolingSystem) SetStatus(ctx context.Context, active bool) error {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    if active == cs.status {
        return nil // Статус не изменился
    }

    if active {
        // Запуск системы охлаждения
        if err := cs.pump.SetSpeed(ctx, 50); err != nil {
            return fmt.Errorf("failed to start pump: %w", err)
        }
        if err := cs.led.SetColor(ctx, 0, 255, 0); err != nil {
            cs.pump.Stop(ctx) // Откат при ошибке
            return fmt.Errorf("failed to set LED: %w", err)
        }
    } else {
        // Остановка системы охлаждения
        if err := cs.pump.Stop(ctx); err != nil {
            return fmt.Errorf("failed to stop pump: %w", err)
        }
        if err := cs.led.Off(ctx); err != nil {
            return fmt.Errorf("failed to turn off LED: %w", err)
        }
    }

    cs.status = active
    return nil
}

func (cs *CoolingSystem) GetStatus(ctx context.Context) (bool, float64, error) {
    cs.mu.RLock()
    defer cs.mu.RUnlock()

    speed, err := cs.pump.GetCurrentSpeed()
    if err != nil {
        return false, 0, fmt.Errorf("failed to get pump speed: %w", err)
    }
    return cs.status, speed, nil
}

// Плавная остановка системы
func (cs *CoolingSystem) Shutdown(ctx context.Context) error {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    // Плавное уменьшение скорости
    currentSpeed, err := cs.pump.GetCurrentSpeed()
    if err != nil {
        return fmt.Errorf("failed to get current speed: %w", err)
    }

    // Красный цвет индикатора во время остановки
    if err := cs.led.SetColor(ctx, 255, 0, 0); err != nil {
        return fmt.Errorf("failed to set warning color: %w", err)
    }

    // Плавное уменьшение скорости
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for speed := currentSpeed; speed > 0; speed -= 2.0 {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := cs.pump.SetSpeed(ctx, speed); err != nil {
                return fmt.Errorf("failed to reduce speed: %w", err)
            }
        }
    }

    // Полное выключение
    if err := cs.pump.Stop(ctx); err != nil {
        return fmt.Errorf("failed to stop pump: %w", err)
    }
    if err := cs.led.Off(ctx); err != nil {
        return fmt.Errorf("failed to turn off LED: %w", err)
    }

    cs.status = false
    return nil
}
```

## Продвинутые примеры использования

### Система управления освещением

```go
// Контроллер группы RGB светодиодов
type LightingController struct {
    pca       *pca9685.PCA9685
    leds      []*pca9685.RGBLed
    mu        sync.RWMutex
    intensity float64
}

func NewLightingController(pca *pca9685.PCA9685, ledGroups [][3]int) (*LightingController, error) {
    lc := &LightingController{
        pca:       pca,
        leds:      make([]*pca9685.RGBLed, 0, len(ledGroups)),
        intensity: 1.0,
    }

    // Инициализация всех групп светодиодов
    for _, channels := range ledGroups {
        led, err := pca9685.NewRGBLed(pca, channels[0], channels[1], channels[2])
        if err != nil {
            return nil, fmt.Errorf("failed to create LED group: %w", err)
        }
        
        // Применение калибровки
        cal := pca9685.RGBCalibration{
            RedMin: 0, RedMax: 4095,
            GreenMin: 0, GreenMax: 3800,
            BlueMin: 100, BlueMax: 4095,
        }
        led.SetCalibration(cal)
        
        lc.leds = append(lc.leds, led)
    }

    return lc, nil
}

// Установка цвета для всех групп с эффектом волны
func (lc *LightingController) SetColorWave(ctx context.Context, targetColor color.Color) error {
    lc.mu.Lock()
    defer lc.mu.Unlock()

    for i, led := range lc.leds {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Установка яркости с учетом позиции в волне
            phase := float64(i) / float64(len(lc.leds))
            brightness := (math.Sin(phase*2*math.Pi) + 1) / 2
            led.SetBrightness(brightness * lc.intensity)
            
            if err := led.SetColorStdlib(ctx, targetColor); err != nil {
                return fmt.Errorf("failed to set LED %d color: %w", i, err)
            }
            
            // Задержка для создания эффекта волны
            time.Sleep(50 * time.Millisecond)
        }
    }
    return nil
}

// Плавное изменение общей яркости
func (lc *LightingController) FadeIntensity(ctx context.Context, target float64, duration time.Duration) error {
    if target < 0 || target > 1 {
        return fmt.Errorf("invalid target intensity: %.2f", target)
    }

    lc.mu.Lock()
    defer lc.mu.Unlock()

    steps := 50
    stepDuration := duration / time.Duration(steps)
    startIntensity := lc.intensity
    
    for i := 0; i <= steps; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            progress := float64(i) / float64(steps)
            newIntensity := startIntensity + (target-startIntensity)*progress
            
            for _, led := range lc.leds {
                led.SetBrightness(newIntensity)
            }
            
            time.Sleep(stepDuration)
        }
    }

    lc.intensity = target
    return nil
}
```

### Система управления потоком с несколькими насосами

```go
// Контроллер группы насосов
type PumpController struct {
    pumps     []*pca9685.Pump
    mu        sync.RWMutex
    status    map[int]float64  // Текущие скорости
}

func NewPumpController(pca *pca9685.PCA9685, channels []int) (*PumpController, error) {
    pc := &PumpController{
        pumps:  make([]*pca9685.Pump, 0, len(channels)),
        status: make(map[int]float64),
    }

    // Инициализация всех насосов
    for _, channel := range channels {
        pump, err := pca9685.NewPump(pca, channel,
            pca9685.WithSpeedLimits(500, 3500))
        if err != nil {
            return nil, fmt.Errorf("failed to create pump on channel %d: %w", channel, err)
        }
        pc.pumps = append(pc.pumps, pump)
        pc.status[channel] = 0
    }

    return pc, nil
}

// Запуск каскадного режима работы насосов
func (pc *PumpController) StartCascade(ctx context.Context, baseSpeed float64) error {
    pc.mu.Lock()
    defer pc.mu.Unlock()

    // Запуск насосов по очереди с разной скоростью
    for i, pump := range pc.pumps {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Вычисление скорости для каждого насоса
            speed := baseSpeed * (1.0 - float64(i)*0.1)
            if err := SoftStart(ctx, pump, speed); err != nil {
                return fmt.Errorf("failed to start pump %d: %w", i, err)
            }
            pc.status[i] = speed
            
            // Задержка между запуском насосов
            time.Sleep(500 * time.Millisecond)
        }
    }
    return nil
}

// Балансировка нагрузки между насосами
func (pc *PumpController) BalanceLoad(ctx context.Context, totalFlow float64) error {
    pc.mu.Lock()
    defer pc.mu.Unlock()

    activeCount := len(pc.pumps)
    if activeCount == 0 {
        return fmt.Errorf("no active pumps")
    }

    // Распределение потока между насосами
    flowPerPump := totalFlow / float64(activeCount)
    
    for i, pump := range pc.pumps {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            currentSpeed, err := pump.GetCurrentSpeed()
            if err != nil {
                return fmt.Errorf("failed to get pump %d speed: %w", i, err)
            }

            // Плавное изменение скорости до целевой
            if err := SoftAdjust(ctx, pump, currentSpeed, flowPerPump); err != nil {
                return fmt.Errorf("failed to adjust pump %d: %w", i, err)
            }
            
            pc.status[i] = flowPerPump
        }
    }
    return nil
}

// Плавная настройка скорости насоса
func SoftAdjust(ctx context.Context, pump *pca9685.Pump, current, target float64) error {
    step := 0.5 // Шаг изменения скорости
    if current < target {
        step = math.Abs(step)
    } else {
        step = -math.Abs(step)
    }

    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for speed := current; ; speed += step {
        // Проверка достижения целевой скорости
        if (step > 0 && speed >= target) || (step < 0 && speed <= target) {
            speed = target
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := pump.SetSpeed(ctx, speed); err != nil {
                return fmt.Errorf("failed to set speed: %w", err)
            }

            // Проверка достижения цели
            if speed == target {
                return nil
            }
        }
    }
}
```

Эти примеры демонстрируют:
1. Потокобезопасную работу с устройствами
2. Корректную обработку ошибок
3. Использование контекстов для отмены операций
4. Плавное управление устройствами
5. Синхронизацию множественных устройств
6. Мониторинг состояния
7. Обработку граничных случаев
