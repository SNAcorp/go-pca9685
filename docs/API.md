### Документация API

#### Интерфейсы

##### I2CDevice
```go
type I2CDevice interface {
    WriteReg(reg uint8, data []byte) error
    ReadReg(reg uint8, data []byte) error
    Close() error
}
```
Базовый интерфейс для работы с I2C устройствами.

#### Конфигурация

##### Config
```go
type Config struct {
    InitialFreq  float64         // Начальная частота PWM (24-1526 Hz)
    InvertLogic  bool           // Инвертировать выходную логику
    OpenDrain    bool           // Использовать open-drain выходы
    Context      context.Context // Контекст для отмены операций
}
```

- `DefaultConfig() *Config` - возвращает конфигурацию по умолчанию

#### PCA9685

##### Основные методы
- `New(dev I2CDevice, config *Config) (*PCA9685, error)` - создает новый контроллер
  - `dev`: I2C устройство
  - `config`: конфигурация (может быть nil)
  - Возвращает: инициализированный контроллер или ошибку

- `Close() error` - освобождает ресурсы и закрывает устройство

- `Reset() error` - сбрасывает настройки контроллера

- `SetPWMFreq(freq float64) error` - устанавливает частоту PWM
  - `freq`: частота в Гц (24-1526)
  - Возвращает: ошибку при выходе за диапазон

##### Управление PWM
- `SetPWM(ctx context.Context, channel int, on, off uint16) error` - прямой контроль PWM
  - `ctx`: контекст для отмены операции
  - `channel`: номер канала (0-15)
  - `on`: время включения (0-4095)
  - `off`: время выключения (0-4095)

- `SetAllPWM(ctx context.Context, on, off uint16) error` - установка всех каналов
  - Параметры аналогичны SetPWM, но применяются ко всем каналам

- `SetMultiPWM(ctx context.Context, settings map[int]struct{On, Off uint16}) error` - групповая установка
  - `settings`: карта каналов и их значений

##### Управление каналами
- `EnableChannels(channels ...int) error` - включает указанные каналы

- `DisableChannels(channels ...int) error` - выключает указанные каналы

- `GetChannelState(channel int) (enabled bool, on uint16, off uint16, err error)` - состояние канала

#### RGBLed

##### Калибровка
```go
type RGBCalibration struct {
    RedMin, RedMax     uint16
    GreenMin, GreenMax uint16
    BlueMin, BlueMax   uint16
}
```

##### Методы
- `NewRGBLed(pca *PCA9685, red, green, blue int) (*RGBLed, error)` - создает RGB светодиод
  - `pca`: контроллер PWM
  - `red, green, blue`: номера каналов (0-15)
  - Возвращает: настроенный светодиод или ошибку

- `SetColor(ctx context.Context, r, g, b uint8) error` - устанавливает цвет
  - `ctx`: контекст для отмены операции
  - `r, g, b`: компоненты цвета (0-255)

- `SetColorStdlib(ctx context.Context, c color.Color) error` - установка через color.Color
  - `c`: цвет из пакета image/color

- `SetBrightness(brightness float64) error` - устанавливает яркость
  - `brightness`: значение от 0.0 до 1.0

- `GetBrightness() float64` - возвращает текущую яркость

- `SetCalibration(cal RGBCalibration)` - устанавливает калибровку
- `GetCalibration() RGBCalibration` - возвращает текущую калибровку

- `Off(ctx context.Context) error` - выключает светодиод

#### Pump

##### Опции конфигурации
- `WithSpeedLimits(min, max uint16) PumpOption` - устанавливает ограничения скорости

##### Методы
- `NewPump(pca *PCA9685, channel int, opts ...PumpOption) (*Pump, error)` - создает контроллер насоса
  - `pca`: контроллер PWM
  - `channel`: номер канала (0-15)
  - `opts`: опциональные настройки
  - Возвращает: настроенный насос или ошибку

- `SetSpeed(ctx context.Context, percent float64) error` - устанавливает скорость
  - `ctx`: контекст для отмены операции
  - `percent`: скорость в процентах (0-100)

- `GetCurrentSpeed() (float64, error)` - возвращает текущую скорость в процентах

- `Stop(ctx context.Context) error` - останавливает насос

- `SetSpeedLimits(min, max uint16) error` - изменяет ограничения скорости
  - `min`: минимальное значение PWM (0-4095)
  - `max`: максимальное значение PWM (0-4095)

### Константы

```go
const (
    MinFrequency  = 24    // Минимальная частота PWM
    MaxFrequency  = 1526  // Максимальная частота PWM
    PwmResolution = 4096  // Разрешение PWM (12 бит)
)
```

### Обработка ошибок

Все методы возвращают ошибки следующих типов:
- `context.Canceled` - операция отменена
- `context.DeadlineExceeded` - превышен таймаут
- Специфические ошибки I2C
- Ошибки валидации параметров

### Потокобезопасность

Все методы библиотеки потокобезопасны и могут вызываться из разных горутин.
При одновременном доступе к устройству используются мьютексы для синхронизации.

### Рекомендации по использованию

1. Всегда используйте контекст для операций, которые могут занять время
2. Проверяйте возвращаемые ошибки
3. Для RGB светодиодов рекомендуется использовать калибровку
4. Для насосов используйте ограничения скорости
5. При групповых операциях предпочитайте SetMultiPWM