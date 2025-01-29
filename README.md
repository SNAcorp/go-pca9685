# PCA9685 Go Library 🔧

**Библиотека на Go для управления PWM-контроллером PCA9685 через I2C.**  
Идеально подходит для управления RGB-светодиодами, сервоприводами, насосами, моторами и другими устройствами в IoT и робототехнике.

[![Go Version](https://img.shields.io/badge/Go-1.20%2B-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Open Source](https://badges.frapsoft.com/os/v2/open-source.svg)](https://opensource.org)

## Особенности ✨
- 🔒 Потокобезопасная реализация с поддержкой конкурентного доступа
- 🎛 Поддержка 16 каналов PWM с разрешением 12 бит (0–4095)
- ✅ Готовые абстракции для RGB-светодиодов и насосов с калибровкой
- ⚡ Автоматическая настройка частоты PWM (24–1526 Гц)
- 🔄 Поддержка контекстов для управления временем жизни операций
- 📖 100% покрытие тестами
- 💪 Отказоустойчивость и защита от некорректных входных данных

## Установка 📦
```bash
go get github.com/SNAcorp/pca9685
```

## Быстрый старт 🚀
### 1. Инициализация контроллера
```go
// Создание I2C устройства
i2c := NewI2CDevice() 

// Настройка конфигурации
config := pca9685.DefaultConfig()
config.InitialFreq = 1000    // 1000 Гц
config.InvertLogic = false   // Прямая логика
config.Context = ctx         // Контекст для отмены операций

// Создание контроллера
pca, err := pca9685.New(i2c, config)
if err != nil {
    log.Fatal(err)
}
defer pca.Close()
```

### 2. Управление RGB-светодиодом
```go
// Создание RGB светодиода на каналах 0,1,2
led, err := pca9685.NewRGBLed(pca, 0, 1, 2)
if err != nil {
    log.Fatal(err)
}

// Калибровка для точной цветопередачи
cal := pca9685.RGBCalibration{
    RedMin: 0, RedMax: 4095,
    GreenMin: 0, GreenMax: 3800,
    BlueMin: 100, BlueMax: 4095,
}
led.SetCalibration(cal)

// Установка цвета и яркости
ctx := context.Background()
led.SetBrightness(0.7)                    // 70% яркости
led.SetColor(ctx, 255, 0, 128)            // Розовый цвет
led.SetColorStdlib(ctx, color.RGBA{...})  // Через стандартный пакет color
```

### 3. Управление насосом
```go
// Создание насоса с ограничениями скорости
pump, err := pca9685.NewPump(pca, 4, 
    pca9685.WithSpeedLimits(500, 3500))
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
// Плавный запуск
pump.SetSpeed(ctx, 85)           // 85% мощности
speed, _ := pump.GetCurrentSpeed() // Получение текущей скорости
pump.Stop(ctx)                   // Остановка
```

### 4. Групповое управление
```go
// Одновременная установка нескольких каналов
settings := map[int]struct{ On, Off uint16 }{
    0: {0, 2048},  // 50% на канале 0
    1: {0, 4095},  // 100% на канале 1
    2: {2048, 4095}, // Особый паттерн на канале 2
}

ctx := context.Background()
pca.SetMultiPWM(ctx, settings)

// Включение/выключение групп каналов
pca.EnableChannels(0, 1, 2)
pca.DisableChannels(3, 4)
```

## Примеры 🧪
- [RGB-светодиод](examples/rgb_led) — анимация с плавными переходами
- [Насос](examples/pump) — плавный запуск и регулировка скорости
- [Групповое управление](examples/multi) — синхронизация нескольких каналов

## Документация 📚
### Основные компоненты
| Компонент       | Описание                                          |
|----------------|--------------------------------------------------|
| `PCA9685`      | Базовый контроллер с поддержкой конкурентности    |
| `RGBLed`       | RGB светодиоды с калибровкой и цветокоррекцией    |
| `Pump`         | Насосы с плавным пуском и ограничением скорости    |
| `Config`       | Гибкая конфигурация с поддержкой контекстов       |

[Полная документация →](docs/API.md)

## Безопасность и надежность 🛡
- ✅ Потокобезопасность через sync.Mutex
- ✅ Контроль времени жизни через context.Context
- ✅ Валидация всех входных параметров
- ✅ Защита от гонок данных
- ✅ Корректная обработка ошибок I2C

## Как внести вклад 🤝
1. Форкните репозиторий
2. Создайте ветку: `git checkout -b feature/your-idea`
3. Запустите тесты: `go test -race ./...`
4. Сделайте коммит: `git commit -m 'Add awesome feature'`
5. Откройте Pull Request

## Совместимость 🔌
**Протестировано на:**
- Raspberry Pi (все модели)
- Orange Pi
- NVIDIA Jetson
- Любые устройства с поддержкой I2C

## Полезные ссылки 🔗
- [Спецификация PCA9685](https://www.nxp.com/docs/en/data-sheet/PCA9685.pdf)
- [Go context package](https://golang.org/pkg/context/)
- [I2C в Go](https://pkg.go.dev/golang.org/x/exp/io/i2c)

## Лицензия ⚖️
MIT License © 2024 [SNAcorp]. См. [LICENSE](LICENSE).

---

**Производительность. Надежность. Простота использования.**