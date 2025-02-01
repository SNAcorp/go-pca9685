# Примеры использования PCA9685

В этом документе собраны различные примеры использования библиотеки PCA9685. От простых случаев до сложных сценариев применения.

## Содержание

1. [Базовая настройка](#базовая-настройка)
2. [Работа с PWM](#работа-с-pwm)
3. [RGB светодиоды](#rgb-светодиоды)
4. [Управление насосами](#управление-насосами)
5. [Расширенные возможности](#расширенные-возможности)
6. [Работа с разными платформами](#работа-с-разными-платформами)
7. [Логирование](#логирование)
8. [Сложные сценарии](#сложные-сценарии)

## Базовая настройка

### Инициализация на Linux с d2r2/go-i2c

```go
package main

import (
    "context"
    "log"
    "github.com/snaart/go-pca9685/pkg/pca9685"
    "github.com/d2r2/go-i2c"
)

func main() {
    // Создание I2C устройства
    i2c, err := i2c.NewI2C(0x40, 1) // 0x40 - адрес устройства, 1 - номер шины I2C
    if err != nil {
        log.Fatalf("Ошибка инициализации I2C: %v", err)
    }
    defer i2c.Close()

    // Создание адаптера
    adapter := pca9685.NewI2CAdapterD2r2(i2c)

    // Настройка конфигурации
    config := pca9685.DefaultConfig()
    config.InitialFreq = 1000    // Частота PWM
    config.InvertLogic = false   // Нормальная логика
    config.OpenDrain = false     // Push-pull выход
    config.LogLevel = pca9685.LogLevelDetailed // Подробное логирование

    // Создание контроллера
    pca, err := pca9685.New(adapter, config)
    if err != nil {
        log.Fatalf("Ошибка создания контроллера: %v", err)
    }
    defer pca.Close()

    log.Println("Контроллер успешно инициализирован")
}
```

### Инициализация на Linux с periph.io

```go
package main

import (
    "log"
    "github.com/snaart/go-pca9685/pkg/pca9685"
    "periph.io/x/conn/v3/i2c"
    "periph.io/x/conn/v3/i2c/i2creg"
    "periph.io/x/host/v3"
)

func main() {
    // Инициализация periph.io
    if _, err := host.Init(); err != nil {
        log.Fatalf("Ошибка инициализации periph.io: %v", err)
    }

    // Открытие шины I2C
    bus, err := i2creg.Open("")
    if err != nil {
        log.Fatalf("Ошибка открытия шины I2C: %v", err)
    }
    defer bus.Close()

    // Создание устройства
    dev := &i2c.Dev{Bus: bus, Addr: 0x40}

    // Создание адаптера
    adapter := pca9685.NewI2CAdapterPeriph(dev)

    // Создание контроллера
    pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
    if err != nil {
        log.Fatalf("Ошибка создания контроллера: %v", err)
    }
    defer pca.Close()

    log.Println("Контроллер успешно инициализирован")
}
```

### Инициализация для тестирования

```go
package main

import (
    "log"
    "github.com/snaart/go-pca9685/pkg/pca9685"
)

func main() {
    // Создание тестового адаптера
    adapter := pca9685.NewTestI2C()

    // Создание контроллера
    pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
    if err != nil {
        log.Fatalf("Ошибка создания контроллера: %v", err)
    }
    defer pca.Close()

    log.Println("Тестовый контроллер успешно инициализирован")
}
```

## Работа с PWM

### Базовое управление PWM

```go
func pwmExample(pca *pca9685.PCA9685) error {
    ctx := context.Background()

    // Установка частоты PWM
    if err := pca.SetPWMFreq(1000); err != nil {
        return err
    }

    // Установка PWM на одном канале
    if err := pca.SetPWM(ctx, 0, 0, 2048); err != nil {
        return err
    }

    // Установка PWM на нескольких каналах
    settings := map[int]struct{ On, Off uint16 }{
        0: {0, 2048},
        1: {0, 1024},
        2: {0, 3072},
    }
    if err := pca.SetMultiPWM(ctx, settings); err != nil {
        return err
    }

    // Установка одинакового значения на всех каналах
    if err := pca.SetAllPWM(ctx, 0, 2048); err != nil {
        return err
    }

    return nil
}
```

### Плавное изменение PWM

```go
func fadeExample(pca *pca9685.PCA9685) error {
    ctx := context.Background()
    
    // Плавное изменение от 0 до максимума за 2 секунды
    if err := pca.FadeChannel(ctx, 0, 0, 4095, 2*time.Second); err != nil {
        return err
    }

    return nil
}
```

## RGB светодиоды

### Создание и управление RGB светодиодом

```go
func rgbExample(pca *pca9685.PCA9685) error {
    ctx := context.Background()

    // Создание RGB светодиода на каналах 0, 1, 2
    led, err := pca9685.NewRGBLed(pca, 0, 1, 2)
    if err != nil {
        return err
    }

    // Установка цвета (R, G, B)
    if err := led.SetColor(ctx, 255, 0, 0); err != nil { // Красный
        return err
    }

    // Использование стандартного пакета color
    color := color.RGBA{R: 0, G: 255, B: 0, A: 255} // Зеленый
    if err := led.SetColorStdlib(ctx, color); err != nil {
        return err
    }

    // Управление яркостью
    if err := led.SetBrightness(0.5); err != nil { // 50% яркости
        return err
    }

    // Калибровка светодиода
    cal := pca9685.RGBCalibration{
        RedMin: 0, RedMax: 4095,
        GreenMin: 0, GreenMax: 3500,
        BlueMin: 0, BlueMax: 3000,
    }
    led.SetCalibration(cal)

    return nil
}
```

### Создание световых эффектов

```go
func rgbEffectsExample(led *pca9685.RGBLed) error {
    ctx := context.Background()

    // Плавная смена цветов
    colors := []struct{ r, g, b uint8 }{
        {255, 0, 0},   // Красный
        {0, 255, 0},   // Зеленый
        {0, 0, 255},   // Синий
        {255, 255, 0}, // Желтый
    }

    for _, c := range colors {
        if err := led.SetColor(ctx, c.r, c.g, c.b); err != nil {
            return err
        }
        time.Sleep(time.Second)
    }

    // Пульсация
    for i := 0; i < 5; i++ {
        // Увеличение яркости
        for b := 0.0; b <= 1.0; b += 0.1 {
            if err := led.SetBrightness(b); err != nil {
                return err
            }
            time.Sleep(50 * time.Millisecond)
        }
        // Уменьшение яркости
        for b := 1.0; b >= 0.0; b -= 0.1 {
            if err := led.SetBrightness(b); err != nil {
                return err
            }
            time.Sleep(50 * time.Millisecond)
        }
    }

    return nil
}
```

## Управление насосами

### Базовое управление насосом

```go
func pumpExample(pca *pca9685.PCA9685) error {
    ctx := context.Background()

    // Создание насоса с ограничениями скорости
    pump, err := pca9685.NewPump(pca, 0, pca9685.WithSpeedLimits(1000, 3000))
    if err != nil {
        return err
    }

    // Установка скорости (0-100%)
    if err := pump.SetSpeed(ctx, 50); err != nil {
        return err
    }

    // Получение текущей скорости
    speed, err := pump.GetCurrentSpeed()
    if err != nil {
        return err
    }
    log.Printf("Текущая скорость: %.1f%%\n", speed)

    // Остановка насоса
    if err := pump.Stop(ctx); err != nil {
        return err
    }

    return nil
}
```

### Плавное управление насосом

```go
func pumpControlExample(pump *pca9685.Pump) error {
    ctx := context.Background()

    // Плавный запуск
    for speed := 0.0; speed <= 100.0; speed += 10.0 {
        if err := pump.SetSpeed(ctx, speed); err != nil {
            return err
        }
        time.Sleep(500 * time.Millisecond)
    }

    // Работа на максимальной скорости
    time.Sleep(5 * time.Second)

    // Плавная остановка
    for speed := 100.0; speed >= 0.0; speed -= 10.0 {
        if err := pump.SetSpeed(ctx, speed); err != nil {
            return err
        }
        time.Sleep(500 * time.Millisecond)
    }

    return nil
}
```

## Расширенные возможности

### Использование контекста для отмены операций

```go
func contextExample(pca *pca9685.PCA9685) error {
    // Создание контекста с таймаутом
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Операция будет отменена, если не завершится за 5 секунд
    if err := pca.SetPWM(ctx, 0, 0, 2048); err != nil {
        if err == context.DeadlineExceeded {
            return fmt.Errorf("операция превысила таймаут")
        }
        return err
    }

    return nil
}
```

### Мониторинг состояния устройства

```go
func monitoringExample(pca *pca9685.PCA9685) {
    // Получение состояния канала
    enabled, on, off, err := pca.GetChannelState(0)
    if err != nil {
        log.Printf("Ошибка получения состояния: %v", err)
        return
    }
    log.Printf("Канал 0: enabled=%v, on=%d, off=%d", enabled, on, off)

    // Вывод полного состояния устройства
    state := pca.DumpState()
    log.Println("Состояние устройства:")
    log.Println(state)
}
```

## Работа с разными платформами

### Создание собственного адаптера

```go
// MyI2CDevice - интерфейс вашего I2C устройства
type MyI2CDevice interface {
    Write(addr uint8, data []byte) error
    Read(addr uint8, count int) ([]byte, error)
    Close() error
}

// MyI2CAdapter - ваш адаптер
type MyI2CAdapter struct {
    dev    MyI2CDevice
    logger pca9685.Logger
}

func NewMyI2CAdapter(dev MyI2CDevice) *MyI2CAdapter {
    return &MyI2CAdapter{
        dev:    dev,
        logger: pca9685.NewDefaultLogger(pca9685.LogLevelBasic),
    }
}

func (a *MyI2CAdapter) WriteReg(reg uint8, data []byte) error {
    a.logger.Detailed("WriteReg: register=0x%X, data=%v", reg, data)
    buf := append([]byte{reg}, data...)
    if err := a.dev.Write(reg, buf); err != nil {
        a.logger.Error("WriteReg error: %v", err)
        return err
    }
    return nil
}

func (a *MyI2CAdapter) ReadReg(reg uint8, data []byte) error {
    a.logger.Detailed("ReadReg: register=0x%X, len=%d", reg, len(data))
    result, err := a.dev.Read(reg, len(data))
    if err != nil {
        a.logger.Error("ReadReg error: %v", err)
        return err
    }
    copy(data, result)
    return nil
}

func (a *MyI2CAdapter) Close() error {
    a.logger.Basic("Closing device")
    return a.dev.Close()
}
```

## Логирование

### Настройка системы логирования

```go
// Создание собственного логгера
type MyLogger struct {
    level pca9685.LogLevel
}

func (l *MyLogger) Basic(msg string, args ...interface{}) {
    log.Printf("[BASIC] "+msg, args...)
}

func (l *MyLogger) Detailed(msg string, args ...interface{}) {
    if l.level >= pca9685.LogLevelDetailed {
        log.Printf("[DETAILED] "+msg, args...)
    }
}

func (l *MyLogger) Error(msg string, args ...interface{}) {
    log.Printf("[ERROR] "+msg, args...)
}

// Пример использования собственного логгера
func loggingExample() {
    // Создание логгера
    myLogger := &MyLogger{
        level: pca9685.LogLevelDetailed,
    }

    // Настройка конфигурации с пользовательским логгером
    config := pca9685.DefaultConfig()
    config.Logger = myLogger
    config.LogLevel = pca9685.LogLevelDetailed

    // Создание контроллера с пользовательским логгером
    adapter := pca9685.NewTestI2C()
    pca, err := pca9685.New(adapter, config)
    if err != nil {
        log.Fatalf("Ошибка создания контроллера: %v", err)
    }

    // Теперь все операции будут использовать ваш логгер
}
```

### Уровни логирования

```go
// Пример разных уровней логирования
func loggingLevelsExample() {
    // Базовый уровень - только важные сообщения
    basicConfig := pca9685.DefaultConfig()
    basicConfig.LogLevel = pca9685.LogLevelBasic

    // Подробный уровень - все сообщения
    detailedConfig := pca9685.DefaultConfig()
    detailedConfig.LogLevel = pca9685.LogLevelDetailed

    // Создание контроллеров с разными уровнями логирования
    adapter1 := pca9685.NewTestI2C()
    adapter2 := pca9685.NewTestI2C()

    pca1, _ := pca9685.New(adapter1, basicConfig)
    pca2, _ := pca9685.New(adapter2, detailedConfig)

    // Логи pca1 будут содержать только базовые сообщения
    // Логи pca2 будут содержать все сообщения, включая отладочные
}
```

## Сложные сценарии

### Управление множеством устройств

```go
// Пример управления несколькими RGB светодиодами
func multipleLedsExample() error {
    ctx := context.Background()
    adapter := pca9685.NewTestI2C()
    pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
    if err != nil {
        return err
    }

    // Создание нескольких RGB светодиодов
    leds := make([]*pca9685.RGBLed, 3)
    for i := 0; i < 3; i++ {
        led, err := pca9685.NewRGBLed(pca, i*3, i*3+1, i*3+2)
        if err != nil {
            return err
        }
        leds[i] = led
    }

    // "Бегущий огонь"
    for {
        for i, led := range leds {
            // Включаем текущий светодиод
            if err := led.SetColor(ctx, 255, 0, 0); err != nil {
                return err
            }

            // Выключаем предыдущий
            prev := (i + len(leds) - 1) % len(leds)
            if err := leds[prev].Off(ctx); err != nil {
                return err
            }

            time.Sleep(500 * time.Millisecond)
        }
    }
}

// Пример управления системой насосов
func multiplePumpsExample() error {
    ctx := context.Background()
    adapter := pca9685.NewTestI2C()
    pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
    if err != nil {
        return err
    }

    // Создание группы насосов
    pumps := make([]*pca9685.Pump, 4)
    for i := 0; i < 4; i++ {
        pump, err := pca9685.NewPump(pca, i, pca9685.WithSpeedLimits(1000, 3000))
        if err != nil {
            return err
        }
        pumps[i] = pump
    }

    // Последовательный запуск насосов
    for i, pump := range pumps {
        speed := float64(25 * (i + 1)) // 25%, 50%, 75%, 100%
        if err := pump.SetSpeed(ctx, speed); err != nil {
            return err
        }
        time.Sleep(time.Second)
    }

    // Работа в течение 10 секунд
    time.Sleep(10 * time.Second)

    // Последовательная остановка
    for i := len(pumps) - 1; i >= 0; i-- {
        if err := pumps[i].Stop(ctx); err != nil {
            return err
        }
        time.Sleep(time.Second)
    }

    return nil
}
```

### Обработка ошибок и восстановление

```go
// Пример обработки ошибок и повторных попыток
func errorHandlingExample(pca *pca9685.PCA9685) error {
    ctx := context.Background()
    maxRetries := 3
    
    // Функция для повторных попыток
    retry := func(operation func() error) error {
        var lastErr error
        for i := 0; i < maxRetries; i++ {
            if err := operation(); err != nil {
                lastErr = err
                log.Printf("Попытка %d не удалась: %v", i+1, err)
                time.Sleep(time.Second * time.Duration(i+1))
                continue
            }
            return nil
        }
        return fmt.Errorf("все попытки не удались, последняя ошибка: %v", lastErr)
    }

    // Пример использования с PWM
    if err := retry(func() error {
        return pca.SetPWM(ctx, 0, 0, 2048)
    }); err != nil {
        return err
    }

    // Пример использования с частотой
    if err := retry(func() error {
        return pca.SetPWMFreq(1000)
    }); err != nil {
        return err
    }

    return nil
}
```

### Синхронизация нескольких устройств

```go
// Пример синхронизации нескольких RGB светодиодов
func synchronizedLedsExample() error {
    ctx := context.Background()
    adapter := pca9685.NewTestI2C()
    pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
    if err != nil {
        return err
    }

    // Создание светодиодов
    led1, err := pca9685.NewRGBLed(pca, 0, 1, 2)
    if err != nil {
        return err
    }
    led2, err := pca9685.NewRGBLed(pca, 3, 4, 5)
    if err != nil {
        return err
    }

    // Использование WaitGroup для синхронизации
    var wg sync.WaitGroup
    
    // Функция для одновременного изменения цвета
    setColorSync := func(r, g, b uint8) error {
        wg.Add(2)
        
        var err1, err2 error
        go func() {
            defer wg.Done()
            err1 = led1.SetColor(ctx, r, g, b)
        }()
        
        go func() {
            defer wg.Done()
            err2 = led2.SetColor(ctx, r, g, b)
        }()
        
        wg.Wait()
        
        if err1 != nil {
            return err1
        }
        if err2 != nil {
            return err2
        }
        return nil
    }

    // Пример синхронизированной анимации
    colors := []struct{ r, g, b uint8 }{
        {255, 0, 0},
        {0, 255, 0},
        {0, 0, 255},
    }

    for _, color := range colors {
        if err := setColorSync(color.r, color.g, color.b); err != nil {
            return err
        }
        time.Sleep(time.Second)
    }

    return nil
}
```

### Профилирование производительности

```go
// Пример профилирования операций
func performanceProfilingExample(pca *pca9685.PCA9685) {
    ctx := context.Background()
    
    // Функция для измерения времени выполнения
    timeTrack := func(start time.Time, name string) {
        elapsed := time.Since(start)
        log.Printf("%s заняло %s", name, elapsed)
    }

    // Профилирование установки PWM
    for i := 0; i < 100; i++ {
        start := time.Now()
        pca.SetPWM(ctx, 0, 0, uint16(i*40))
        timeTrack(start, "SetPWM")
        time.Sleep(10 * time.Millisecond)
    }

    // Профилирование множественной установки PWM
    settings := make(map[int]struct{ On, Off uint16 }, 16)
    for i := 0; i < 16; i++ {
        settings[i] = struct{ On, Off uint16 }{0, uint16(i * 250)}
    }

    start := time.Now()
    pca.SetMultiPWM(ctx, settings)
    timeTrack(start, "SetMultiPWM")
}
```

## Многопоточное программирование

### Базовые принципы многопоточной работы

1. **Синхронизация доступа:**
   - Использование мьютексов для защиты общих ресурсов
   - Применение RWMutex для оптимизации параллельного чтения
   - Контекст для отмены операций
   - WaitGroup для синхронизации горутин

2. **Обработка ошибок:**
   - Сбор ошибок из всех горутин
   - Безопасное завершение при ошибках
   - Корректное освобождение ресурсов

### Примеры многопоточного использования

#### Параллельное управление RGB светодиодами

```go
func parallelRGBControl() error {
    // Инициализация контроллера
    pca, err := initController()
    if err != nil {
        return err
    }
    defer pca.Close()

    // Создание нескольких RGB светодиодов
    leds := make([]*RGBLed, 4)
    for i := 0; i < 4; i++ {
        led, err := NewRGBLed(pca, i*3, i*3+1, i*3+2)
        if err != nil {
            return err
        }
        leds[i] = led
    }

    // Создание контекста с отменой
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // WaitGroup для синхронизации горутин
    var wg sync.WaitGroup
    // Канал для сбора ошибок
    errChan := make(chan error, len(leds))

    // Запуск горутин для каждого светодиода
    for i, led := range leds {
        wg.Add(1)
        go func(id int, l *RGBLed) {
            defer wg.Done()

            // Бесконечная анимация цвета для каждого светодиода
            for {
                select {
                case <-ctx.Done():
                    return
                default:
                    // Вычисление цвета на основе времени и ID светодиода
                    t := time.Now().UnixNano() / int64(time.Millisecond)
                    r := uint8(math.Sin(float64(t+id*1000)/1000.0)*127 + 128)
                    g := uint8(math.Sin(float64(t+id*1000)/1000.0+2)*127 + 128)
                    b := uint8(math.Sin(float64(t+id*1000)/1000.0+4)*127 + 128)

                    if err := l.SetColor(ctx, r, g, b); err != nil {
                        errChan <- fmt.Errorf("LED %d error: %v", id, err)
                        return
                    }

                    time.Sleep(50 * time.Millisecond)
                }
            }
        }(i, led)
    }

    // Горутина для мониторинга ошибок
    go func() {
        for err := range errChan {
            log.Printf("Error detected: %v", err)
            cancel() // Отмена всех операций при ошибке
        }
    }()

    // Ожидание завершения всех горутин
    wg.Wait()
    close(errChan)

    return nil
}
```

#### Параллельное управление насосами

```go
func parallelPumpControl() error {
    pca, err := initController()
    if err != nil {
        return err
    }
    defer pca.Close()

    // Создание группы насосов
    pumps := make([]*Pump, 4)
    for i := 0; i < 4; i++ {
        pump, err := NewPump(pca, i, WithSpeedLimits(1000, 3000))
        if err != nil {
            return err
        }
        pumps[i] = pump
    }

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
    defer cancel()

    var wg sync.WaitGroup
    errChan := make(chan error, len(pumps))

    // Запуск горутин для каждого насоса
    for i, pump := range pumps {
        wg.Add(1)
        go func(id int, p *Pump) {
            defer wg.Done()

            // Различные паттерны работы для разных насосов
            switch id % 4 {
            case 0: // Плавное изменение скорости
                for {
                    select {
                    case <-ctx.Done():
                        return
                    default:
                        for speed := 0.0; speed <= 100.0; speed += 10.0 {
                            if err := p.SetSpeed(ctx, speed); err != nil {
                                errChan <- fmt.Errorf("pump %d error: %v", id, err)
                                return
                            }
                            time.Sleep(500 * time.Millisecond)
                        }
                        for speed := 100.0; speed >= 0.0; speed -= 10.0 {
                            if err := p.SetSpeed(ctx, speed); err != nil {
                                errChan <- fmt.Errorf("pump %d error: %v", id, err)
                                return
                            }
                            time.Sleep(500 * time.Millisecond)
                        }
                    }
                }

            case 1: // Импульсный режим
                for {
                    select {
                    case <-ctx.Done():
                        return
                    default:
                        if err := p.SetSpeed(ctx, 100); err != nil {
                            errChan <- err
                            return
                        }
                        time.Sleep(2 * time.Second)
                        if err := p.SetSpeed(ctx, 0); err != nil {
                            errChan <- err
                            return
                        }
                        time.Sleep(1 * time.Second)
                    }
                }

            case 2: // Случайные скорости
                for {
                    select {
                    case <-ctx.Done():
                        return
                    default:
                        speed := rand.Float64() * 100
                        if err := p.SetSpeed(ctx, speed); err != nil {
                            errChan <- err
                            return
                        }
                        time.Sleep(time.Second)
                    }
                }

            case 3: // Ступенчатое изменение
                speeds := []float64{25, 50, 75, 100}
                for {
                    select {
                    case <-ctx.Done():
                        return
                    default:
                        for _, speed := range speeds {
                            if err := p.SetSpeed(ctx, speed); err != nil {
                                errChan <- err
                                return
                            }
                            time.Sleep(3 * time.Second)
                        }
                    }
                }
            }
        }(i, pump)
    }

    // Горутина для мониторинга состояния
    wg.Add(1)
    go func() {
        defer wg.Done()
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                for i, pump := range pumps {
                    speed, err := pump.GetCurrentSpeed()
                    if err != nil {
                        log.Printf("Error getting pump %d speed: %v", i, err)
                        continue
                    }
                    log.Printf("Pump %d speed: %.2f%%", i, speed)
                }
            }
        }
    }()

    // Ожидание ошибки или завершения по таймауту
    select {
    case err := <-errChan:
        cancel()
        wg.Wait()
        return fmt.Errorf("pump control error: %v", err)
    case <-ctx.Done():
        wg.Wait()
        return ctx.Err()
    }
}
```

#### Синхронизированное управление устройствами

```go
func synchronizedDeviceControl() error {
    pca, err := initController()
    if err != nil {
        return err
    }
    defer pca.Close()

    // Создание RGB светодиода и насоса
    led, err := NewRGBLed(pca, 0, 1, 2)
    if err != nil {
        return err
    }

    pump, err := NewPump(pca, 3)
    if err != nil {
        return err
    }

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()

    var wg sync.WaitGroup
    errChan := make(chan error, 2)

    // Синхронизация через канал
    syncChan := make(chan struct{})

    // Управление светодиодом
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            select {
            case <-ctx.Done():
                return
            case <-syncChan:
                // Изменение цвета в зависимости от скорости насоса
                speed, err := pump.GetCurrentSpeed()
                if err != nil {
                    errChan <- err
                    return
                }

                // Цвет от синего (низкая скорость) до красного (высокая)
                r := uint8(speed * 2.55)
                b := uint8(255 - (speed * 2.55))
                if err := led.SetColor(ctx, r, 0, b); err != nil {
                    errChan <- err
                    return
                }
            }
        }
    }()

    // Управление насосом
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // Плавное изменение скорости
                for speed := 0.0; speed <= 100.0; speed += 5.0 {
                    if err := pump.SetSpeed(ctx, speed); err != nil {
                        errChan <- err
                        return
                    }
                    // Сигнал светодиоду об изменении скорости
                    syncChan <- struct{}{}
                    time.Sleep(200 * time.Millisecond)
                }
                for speed := 100.0; speed >= 0.0; speed -= 5.0 {
                    if err := pump.SetSpeed(ctx, speed); err != nil {
                        errChan <- err
                        return
                    }
                    // Сигнал светодиоду об изменении скорости
                    syncChan <- struct{}{}
                    time.Sleep(200 * time.Millisecond)
                }
            }
        }
    }()

    // Ожидание завершения или ошибки
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case err := <-errChan:
        cancel()
        return fmt.Errorf("device control error: %v", err)
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

Эти примеры демонстрируют различные аспекты работы с библиотекой PCA9685, от базового использования до сложных сценариев применения. Все примеры включают обработку ошибок и следуют лучшим практикам программирования на Go.

Для получения дополнительной информации обратитесь к документации пакета или к тестам в файле `pca9685_test.go`.