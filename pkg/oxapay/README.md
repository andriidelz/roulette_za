# OxaPay Go Client

Клиент для интеграции платежной системы OxaPay в Go приложения.

## Установка

```bash
go get github.com/example/oxapay
```

## Использование

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/example/oxapay"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func main() {
    // Подключение к базе данных
    db := ConnectToDatabase()
    
    // Инициализация роутера
    router := gin.Default()
    
    // Создание конфигурации OxaPay
    config := oxapay.Config{
        APIKey:      "your-api-key",
        WebhookKey:  "your-webhook-key",
        CallbackURL: "https://your-domain.com/webhooks/oxapay",
        DB:          db,
    }
    
    // Создание клиента OxaPay
    client := oxapay.NewClient(config)
    
    // Инициализация таблиц в БД
    if err := client.InitializeTables(); err != nil {
        log.Fatalf("Failed to initialize OxaPay tables: %v", err)
    }
    
    // Настройка вебхуков
    client.SetupWebhookHandler(router, "/webhooks/oxapay")
    
    // Пример создания выплаты
    payout, err := client.CreatePayout(oxapay.PayoutRequest{
        Currency:    "USDT",
        Amount:      100.0,
        Address:     "TRX-wallet-address",
        Network:     "TRC20",
        CallbackURL: "https://your-domain.com/webhooks/oxapay",
        Description: "Withdrawal for user 123",
        UserID:      "123",
    })
    
    if err != nil {
        log.Fatalf("Failed to create payout: %v", err)
    }
    
    fmt.Printf("Created payout: %s with status: %s\n", payout.ID, payout.Status)
    
    // Получение информации о выплате
    payoutInfo, err := client.GetPayout(payout.ID)
    if err != nil {
        log.Fatalf("Failed to get payout info: %v", err)
    }
    
    fmt.Printf("Payout status: %s, Transaction hash: %s\n", 
        payoutInfo.Status, payoutInfo.TransactionHash)
    
    // Получение истории выплат
    params := map[string]string{
        "status":   "processing",
        "currency": "USDT",
        "size":     "10",
        "page":     "1",
    }
    
    payouts, meta, err := client.GetPayoutHistory(params)
    if err != nil {
        log.Fatalf("Failed to get payout history: %v", err)
    }
    
    fmt.Printf("Found %d payouts out of %d total\n", len(payouts), meta.Total)
    
    // Запуск сервера
    router.Run(":8080")
}
```

## Структура клиента

Клиент представляет собой обертку над API OxaPay со следующими возможностями:

1. Создание выплат (Create Payout)
2. Проверка статуса выплат (Get Payout)
3. Получение истории выплат (Get Payout History)
4. Обработка вебхуков с подтверждением подписи
5. Хранение данных о выплатах в базе данных

## API методы

### CreatePayout

Создает новую выплату и возвращает объект Payout с информацией о созданной транзакции.

```go
payout, err := client.CreatePayout(oxapay.PayoutRequest{
    Currency:    "BTC",
    Amount:      0.05,
    Address:     "wallet-address",
    Network:     "Bitcoin Network", // опционально, для мультисетевых валют
    CallbackURL: "https://your-domain.com/webhooks/oxapay",
    Description: "Withdrawal",
    Memo:        "Optional memo", // опционально, для сетей с поддержкой мемо (TON и др.)
})
```

### GetPayout

Получает информацию о выплате по её ID.

```go
payout, err := client.GetPayout("track_id")
```

### GetPayoutHistory

Получает историю выплат с возможностью фильтрации.

```go
params := map[string]string{
    "status":      "processing", // статус транзакций
    "currency":    "USDT",       // криптовалюта
    "network":     "TRC20",      // сеть
    "from_amount": "10",         // минимальная сумма
    "to_amount":   "1000",       // максимальная сумма
    "from_date":   "1633046400", // начало периода (unix timestamp)
    "to_date":     "1635724800", // конец периода (unix timestamp)
    "sort_by":     "create_date", // сортировка (create_date, pay_date, amount)
    "sort_type":   "desc",        // порядок сортировки (asc, desc)
    "size":        "20",          // размер страницы (1-200)
    "page":        "1",           // номер страницы
}

payouts, meta, err := client.GetPayoutHistory(params)
```

## Типы статусов выплат

- `processing` - Запрос отправлен и обрабатывается
- `pending` - Запрос обработан и находится в очереди на оплату
- `confirming` - Транзакция создана и ожидает подтверждения в блокчейне
- `confirmed` - Транзакция успешно оплачена
- `canceled` - Запрос на выплату был отменен
- `rejected` - Запрос был отклонен по каким-либо причинам

## Безопасность

Для обеспечения безопасности вебхуков используется проверка подписи (HMAC-SHA256). Убедитесь, что вы указали правильный WebhookKey в конфигурации.

## Требования

- Go 1.18+
- GORM
- Gin (для вебхуков)
- PostgreSQL или другая поддерживаемая GORM БД

## Лицензия

MIT
