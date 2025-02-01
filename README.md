# PCA9685 Go Driver

Драйвер для управления 16-канальным PWM-контроллером PCA9685 на языке Go. Поддерживает работу с RGB светодиодами, насосами и другими PWM-устройствами.

## Особенности

- 🔧 Поддержка 16 независимых PWM-каналов
- 🌈 Встроенная поддержка RGB светодиодов
- 💧 Управление насосами с контролем скорости
- 🔌 Гибкая система адаптеров для разных I²C библиотек
- 🧪 Встроенный эмулятор для тестирования
- 📝 Расширенное логирование
- 🔄 Поддержка контекстов для отмены операций
- 🔒 Потокобезопасность

## Установка

```bash
go get github.com/snaart/go-pca9685
```

## Быстрый старт

### Базовое использование

```go
package main

import (
    "context"
    "github.com/snaart/go-pca9685/pkg/pca9685"
    "github.com/d2r2/go-i2c"
)

func main() {
    // Инициализация I2C
    i2c, err := i2c.NewI2C(0x40, 1)
    if err != nil {
        panic(err)
    }
    defer i2c.Close()

    // Создание адаптера
    adapter := pca9685.NewI2CAdapterD2r2(i2c)

    // Инициализация контроллера
    config := pca9685.DefaultConfig()
    pca, err := pca9685.New(adapter, config)
    if err != nil {
        panic(err)
    }
    defer pca.Close()

    // Установка PWM на канале 0
    ctx := context.Background()
    err = pca.SetPWM(ctx, 0, 0, 2048)
}
```

### Работа с RGB светодиодом

```go
// Создание RGB светодиода на каналах 0, 1, 2
led, err := pca9685.NewRGBLed(pca, 0, 1, 2)
if err != nil {
    panic(err)
}

// Установка цвета
ctx := context.Background()
err = led.SetColor(ctx, 255, 0, 0) // Красный цвет
```

### Управление насосом

```go
// Создание насоса на канале 3
pump, err := pca9685.NewPump(pca, 3, pca9685.WithSpeedLimits(1000, 3000))
if err != nil {
    panic(err)
}

// Установка скорости 50%
ctx := context.Background()
err = pump.SetSpeed(ctx, 50)
```

### Расширенные примеры/сценарии: [Примеры](./examples/examples.md).

## Архитектура

### Система адаптеров

Проект использует систему адаптеров для абстрагирования работы с различными I²C библиотеками:

1. **I2CAdapterD2r2** - адаптер для библиотеки d2r2/go-i2c
2. **I2CAdapterPeriph** - адаптер для библиотеки periph.io
3. **TestI2C** - адаптер-эмулятор для тестирования

Все адаптеры реализуют интерфейс `I2C`:

```go
type I2C interface {
    WriteReg(reg uint8, data []byte) error
    ReadReg(reg uint8, data []byte) error
    Close() error
}
```

### Создание собственного адаптера

Для создания собственного адаптера необходимо реализовать интерфейс `I2C`. Пример:

```go
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
    // Реализация записи в регистр
}

func (a *MyI2CAdapter) ReadReg(reg uint8, data []byte) error {
    // Реализация чтения из регистра
}

func (a *MyI2CAdapter) Close() error {
    // Реализация закрытия устройства
}
```

## Система логирования

Проект использует гибкую систему логирования с двумя уровнями:

- **LogLevelBasic** - только основные события
- **LogLevelDetailed** - подробное логирование всех операций

Логгер можно настроить при создании контроллера:

```go
config := pca9685.DefaultConfig()
config.LogLevel = pca9685.LogLevelDetailed
config.Logger = MyCustomLogger{} // Должен реализовывать интерфейс Logger
```

Интерфейс Logger:

```go
type Logger interface {
    Basic(msg string, args ...interface{})
    Detailed(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

## Тестирование

### Эмулятор TestI2C

Для тестирования без реального оборудования используется `TestI2C`:

```go
// Создание эмулятора
adapter := pca9685.NewTestI2C()

// Создание контроллера с эмулятором
pca, err := pca9685.New(adapter, pca9685.DefaultConfig())
```

Эмулятор сохраняет все записи в память и эмулирует чтение/запись регистров.

### Запуск тестов

```bash
go test ./... -v
```

### Расширенная документация: [Docs](./docs/API.md).

## Поддерживаемые платформы

### Linux

- Полная поддержка через d2r2/go-i2c и periph.io
- Доступ к аппаратному I²C

### MacOS/Windows

- Поддержка через эмулятор TestI2C
- Идеально подходит для разработки и тестирования

## Как внести вклад 🤝
1. Форкните репозиторий
2. Создайте ветку: `git checkout -b feature/your-idea`
3. Запустите тесты: `go test -race ./...`
4. Сделайте коммит: `git commit -m 'Add awesome feature'`
5. Откройте Pull Request

## Рекомендации по внесению изменений

1. Следуйте стилю кода Go
2. Добавляйте тесты для новой функциональности
3. Обновляйте документацию
4. Используйте информативные сообщения коммитов

### Структура PR

- Описание изменений
- Причина изменений
- Тесты
- Примеры использования


## Структура проекта

```
.
├── adapter_d2r2_linux.go    // Адаптер для d2r2/go-i2c
├── adapter_periph_io_linux.go // Адаптер для periph.io
├── adapter_testing.go       // Тестовый адаптер
├── logger.go               // Система логирования
├── pca9685.go             // Основной код контроллера
├── pump.go                // Управление насосами
├── rgb.go                 // Управление RGB светодиодами
└── pca9685_test.go       // Тесты
```

## Полезные ссылки 🔗
- [Спецификация PCA9685](https://www.nxp.com/docs/en/data-sheet/PCA9685.pdf)
- [Go context package](https://golang.org/pkg/context/)
- [I2C в Go](https://pkg.go.dev/golang.org/x/exp/io/i2c)

## Лицензия ⚖️
MIT License © 2024 [SNAcorp]. См. [LICENSE](LICENSE).

---

**Производительность. Надежность. Простота использования.**
