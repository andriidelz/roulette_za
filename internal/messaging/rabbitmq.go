package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Типы сообщений
	EventRoundCompleted = "round_completed"
	EventRoundStarted   = "round_started"
	EventBetPlaced      = "bet_placed"
	EventUserNotified   = "user_notified"

	// Ключи маршрутизации
	RoutingRoundCompleted = "round.completed"
	RoutingRoundStarted   = "round.started"
	RoutingBetPlaced      = "bet.placed"
	RoutingUserNotified   = "user.notified"

	// Приоритеты сообщений
	PriorityLow    = 1
	PriorityNormal = 5
	PriorityHigh   = 7
	PriorityMax    = 9
)

// RouletteMessage определяет структуру сообщений обмена
type RouletteMessage struct {
	Type            string      `json:"type"`             // Тип сообщения (round_completed, round_started и т.д.)
	RoundID         uint        `json:"round_id"`         // ID раунда
	Data            interface{} `json:"data,omitempty"`   // Произвольные данные
	Timestamp       int64       `json:"timestamp"`        // Время создания сообщения
	Sequence        int         `json:"sequence"`         // Порядковый номер для обеспечения правильной последовательности
	SourceComponent string      `json:"source_component"` // Компонент-источник (rotator, bot, admin)
}

// MessageConfig содержит конфигурацию для публикации сообщения
type MessageConfig struct {
	RoutingKey string
	EventType  string
	Priority   uint8
}

// RabbitMQ - клиент для работы с RabbitMQ
type RabbitMQ struct {
	conn          *amqp.Connection
	ch            *amqp.Channel
	connURL       string
	exchangeName  string
	componentName string
	seqCounter    int
	mu            sync.Mutex
	isConnected   bool
}

// NewRabbitMQ создает новый экземпляр клиента RabbitMQ
func NewRabbitMQ(connURL, exchangeName, componentName string) (*RabbitMQ, error) {
	client := &RabbitMQ{
		connURL:       connURL,
		exchangeName:  exchangeName,
		componentName: componentName,
		seqCounter:    0,
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	// Запускаем горутину, которая будет проверять соединение и переподключаться при разрыве
	go client.reconnectLoop()

	return client, nil
}

// Connect устанавливает соединение с RabbitMQ и настраивает обменник
func (r *RabbitMQ) Connect() error {
	var err error

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isConnected {
		return nil
	}

	// Подключаемся к RabbitMQ
	r.conn, err = amqp.Dial(r.connURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Открываем канал
	r.ch, err = r.conn.Channel()
	if err != nil {
		r.conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// Настраиваем обменник
	err = r.ch.ExchangeDeclare(
		r.exchangeName, // имя обменника
		"topic",        // тип обменника (topic для маршрутизации по ключам)
		true,           // долговечный (сохраняется при перезапуске)
		false,          // не автоудаляемый
		false,          // не внутренний
		false,          // не ждать подтверждения от сервера
		nil,            // дополнительные аргументы
	)
	if err != nil {
		r.ch.Close()
		r.conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	r.isConnected = true
	log.Printf("[%s] Connected to RabbitMQ and set up exchange '%s'", r.componentName, r.exchangeName)
	return nil
}

// Close закрывает соединение с RabbitMQ
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isConnected {
		return nil
	}

	if r.ch != nil {
		r.ch.Close()
	}

	if r.conn != nil {
		r.conn.Close()
	}

	r.isConnected = false
	return nil
}

// Publish публикует сообщение в очередь
func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, msgType string, roundID uint, data interface{}, priority uint8) error {
	r.mu.Lock()
	// Увеличиваем счетчик последовательности
	r.seqCounter++
	seq := r.seqCounter
	r.mu.Unlock()

	// Если нет соединения, пытаемся подключиться
	if !r.isConnected {
		if err := r.Connect(); err != nil {
			return fmt.Errorf("cannot publish, not connected: %w", err)
		}
	}

	// Создаем объект сообщения
	message := RouletteMessage{
		Type:            msgType,
		RoundID:         roundID,
		Data:            data,
		Timestamp:       time.Now().UnixNano(),
		Sequence:        seq,
		SourceComponent: r.componentName,
	}

	// Кодируем сообщение в JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	// Публикуем сообщение в очередь с таймаутом через контекст
	r.mu.Lock()
	defer r.mu.Unlock()

	err = r.ch.PublishWithContext(
		ctx,
		r.exchangeName, // имя обменника
		routingKey,     // ключ маршрутизации (например, "round.completed")
		false,          // не обязательно (если не удалось доставить, сообщение будет отброшено)
		false,          // не срочно
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // сообщение сохраняется при перезапуске брокера
			Priority:     priority,        // приоритет сообщения (0-9, 9 - самый высокий)
			Timestamp:    time.Now(),      // время отправки
		},
	)

	if err != nil {
		// Помечаем соединение как разорванное для последующего переподключения
		r.isConnected = false
		return fmt.Errorf("error publishing message: %w", err)
	}

	log.Printf("[%s] Published message: type=%s, round_id=%d, routing_key=%s, seq=%d, priority=%d",
		r.componentName, msgType, roundID, routingKey, seq, priority)
	return nil
}

// PublishWithConfig публикует сообщение с использованием конфигурации
func (r *RabbitMQ) PublishWithConfig(ctx context.Context, config MessageConfig, roundID uint, data interface{}) error {
	return r.Publish(ctx, config.RoutingKey, config.EventType, roundID, data, config.Priority)
}

// PublishRoundCompleted публикует сообщение о завершении раунда
func (r *RabbitMQ) PublishRoundCompleted(ctx context.Context, roundID uint, data interface{}) error {
	config := MessageConfig{
		RoutingKey: RoutingRoundCompleted,
		EventType:  EventRoundCompleted,
		Priority:   PriorityMax,
	}
	return r.PublishWithConfig(ctx, config, roundID, data)
}

// PublishRoundStarted публикует сообщение о начале нового раунда
func (r *RabbitMQ) PublishRoundStarted(ctx context.Context, roundID uint, data interface{}) error {
	config := MessageConfig{
		RoutingKey: RoutingRoundStarted,
		EventType:  EventRoundStarted,
		Priority:   PriorityHigh,
	}
	return r.PublishWithConfig(ctx, config, roundID, data)
}

// PublishBetPlaced публикует сообщение о размещении ставки
func (r *RabbitMQ) PublishBetPlaced(ctx context.Context, roundID uint, data interface{}) error {
	config := MessageConfig{
		RoutingKey: RoutingBetPlaced,
		EventType:  EventBetPlaced,
		Priority:   PriorityNormal,
	}
	return r.PublishWithConfig(ctx, config, roundID, data)
}

// PublishUserNotified публикует сообщение об уведомлении пользователя
func (r *RabbitMQ) PublishUserNotified(ctx context.Context, roundID uint, data interface{}) error {
	config := MessageConfig{
		RoutingKey: RoutingUserNotified,
		EventType:  EventUserNotified,
		Priority:   PriorityLow,
	}
	return r.PublishWithConfig(ctx, config, roundID, data)
}

// SubscribeToQueue подписывается на сообщения из указанной очереди
func (r *RabbitMQ) SubscribeToQueue(queueName string, routingKeys []string, handler func(message RouletteMessage) error) error {
	// Проверяем соединение
	if !r.isConnected {
		if err := r.Connect(); err != nil {
			return fmt.Errorf("cannot subscribe, not connected: %w", err)
		}
	}

	// Создаем очередь с настройкой приоритетов
	args := amqp.Table{
		"x-max-priority": 10, // Максимальный приоритет 10
	}

	// Создаем очередь, если она не существует
	queue, err := r.ch.QueueDeclare(
		queueName, // имя очереди
		true,      // долговечная (сохраняется при перезапуске)
		false,     // не удалять, когда нет подписчиков
		false,     // не эксклюзивная (может использоваться разными соединениями)
		false,     // не ждать подтверждения от сервера
		args,      // аргументы с настройкой приоритетов
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Привязываем очередь к обменнику по каждому ключу маршрутизации
	for _, key := range routingKeys {
		err = r.ch.QueueBind(
			queue.Name,     // имя очереди
			key,            // ключ маршрутизации
			r.exchangeName, // имя обменника
			false,          // не ждать подтверждения от сервера
			nil,            // дополнительные аргументы
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue to key '%s': %w", key, err)
		}
		log.Printf("[%s] Bound queue '%s' to exchange '%s' with routing key '%s'",
			r.componentName, queue.Name, r.exchangeName, key)
	}

	// Устанавливаем QoS для ограничения числа необработанных сообщений
	err = r.ch.Qos(
		1,     // prefetch count - получать только одно сообщение за раз
		0,     // prefetch size - игнорировать
		false, // global - настройки применяются только к этому каналу
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Подписываемся на сообщения
	msgs, err := r.ch.Consume(
		queue.Name, // имя очереди
		"",         // consumer tag - генерируется автоматически
		false,      // auto-ack - не подтверждать автоматически (подтверждение вручную)
		false,      // exclusive - не эксклюзивно
		false,      // no-local - получать все сообщения, включая свои собственные
		false,      // no-wait - ждать подтверждения от сервера
		nil,        // дополнительные аргументы
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Запускаем обработчик сообщений в отдельной горутине
	go r.messageConsumer(msgs, handler, queue.Name)

	log.Printf("[%s] Subscribed to queue '%s' with %d routing keys", r.componentName, queueName, len(routingKeys))
	return nil
}

// messageConsumer обрабатывает сообщения из канала
func (r *RabbitMQ) messageConsumer(msgs <-chan amqp.Delivery, handler func(message RouletteMessage) error, queueName string) {
	for msg := range msgs {
		// Декодируем сообщение
		var message RouletteMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("[%s] Error decoding message: %v", r.componentName, err)
			msg.Nack(false, true) // отказ от сообщения с повторной доставкой
			continue
		}

		log.Printf("[%s] Received message: type=%s, round_id=%d, routing_key=%s, seq=%d, priority=%d from %s",
			r.componentName, message.Type, message.RoundID, msg.RoutingKey, message.Sequence, msg.Priority, message.SourceComponent)

		// Обрабатываем сообщение через предоставленный обработчик
		if err := handler(message); err != nil {
			log.Printf("[%s] Error handling message: %v", r.componentName, err)
			// При ошибке обработки не подтверждаем получение, чтобы сообщение было повторно доставлено
			msg.Nack(false, true)
			continue
		}

		// Подтверждаем успешную обработку сообщения
		msg.Ack(false)
	}

	log.Printf("[%s] Subscription to queue '%s' was closed", r.componentName, queueName)
}

// reconnectLoop проверяет состояние соединения и пытается переподключиться при разрыве
func (r *RabbitMQ) reconnectLoop() {
	for {
		// Проверяем соединение каждые 5 секунд
		time.Sleep(5 * time.Second)

		r.mu.Lock()
		needsReconnect := !r.isConnected || r.conn == nil || r.conn.IsClosed() || r.ch == nil
		r.mu.Unlock()

		if needsReconnect {
			log.Printf("[%s] Connection lost, attempting to reconnect...", r.componentName)
			for i := 0; i < 5; i++ { // Пробуем переподключиться 5 раз
				if err := r.Close(); err != nil {
					log.Printf("[%s] Error closing connection: %v", r.componentName, err)
				}

				if err := r.Connect(); err != nil {
					log.Printf("[%s] Reconnect attempt %d failed: %v", r.componentName, i+1, err)
					time.Sleep(2 * time.Second) // Ждем перед следующей попыткой
					continue
				}

				log.Printf("[%s] Successfully reconnected", r.componentName)
				break
			}
		}
	}
}
