package bot

import (
	"context"
	"fmt"
	"os"
	"roulette/internal/logger"
	"roulette/internal/models"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

const (
	sendMessage      = "sendMessage"
	editMessageText  = "editMessageText"
	sendPhoto        = "sendPhoto"
	sendCaptcha      = "sendCaptcha"
	sendVideo        = "sendVideo"
	editMessageMedia = "editMessageMedia"
	sendSticker      = "sendSticker"
	sendAnimation    = "sendAnimation"
)

// MessageOptions содержит опции для отправки или обновления сообщения
type MessageOptions struct {
	// Время создания сообщения Unix (Нужно чтобы sorted set не перезаписывал сообщение)
	CreatedAt int64
	// Время жизни сообщения, если не указано будет взято из const userQueueExpiration
	// Используется для определения очередности доставки сообщения!
	// Если указать время то внутри очереди пользователя на отправку оно переместися
	// ORDER отправки == TTL
	TTL time.Duration

	// MethodName - Метод телеграма
	MethodName string
	Type       string

	// MessageID - ID сообщения для изменения или удаления
	MessageID int

	// Text - текст сообщения
	Text string

	// PhotoPath - путь к фото (если установлен, будет отправлено фото с подписью Text)
	PhotoPath string
	DelPhoto  bool // true для временных фото, после отправки запускается удаление через какое то время после отправки
	// PhotoFileID - FileID фото (если установлен, будет отправлено фото с подписью Text)
	PhotoFileID string

	// VideoPath - путь к видео (если установлен, будет отправлено видео с подписью Text)
	VideoPath string
	// VideoFileID - FileID видео (если установлен, будет отправлено видео с подписью Text)
	VideoFileID string

	// InlineKeyboard - инлайн клавиатура (если установлена, будет добавлена к сообщению)
	InlineKeyboard *telego.InlineKeyboardMarkup

	// ReplyKeyboard - клавиатура ответа (если установлена, будет добавлена к сообщению)
	ReplyKeyboard *telego.ReplyKeyboardMarkup

	// RemoveKeyboard - если true, клавиатура ответа будет удалена
	RemoveKeyboard bool

	// OneTimeKeyboard - если true и установлено ReplyKeyboard, клавиатура будет одноразовой
	OneTimeKeyboard bool

	// Selective - если true и установлено ReplyKeyboard, клавиатура будет показана только определенным пользователям
	Selective bool

	// ParseMode - режим форматирования текста (HTML, Markdown, MarkdownV2)
	ParseMode string

	// Entities - специальные сущности (эмодзи, форматирование и т.д.)
	Entities []telego.MessageEntity

	// DisableWebPagePreview - если true, превью ссылок будет отключено
	DisableWebPagePreview bool

	// DisableNotification - если true, сообщение будет отправлено беззвучно
	DisableNotification bool
}

// prepareMessage - подготовка сообщения - установка текста и фото/видео если указаны
func (b *Bot) prepareMessage(key, languageCode string) (options MessageOptions) {
	var res models.Localization

	b.localMutex.Lock()
	localizationMap, ok := b.localizations[languageCode]
	if ok {
		res = localizationMap[key]
	}
	b.localMutex.Unlock()

	// Значення не знайдено, пробуємо отримати з бази
	if res.Value == "" {
		logger.Info.Println("Key not found: ", key, languageCode)
		res, _ = b.service.GetRepo().GetLocalization(key, languageCode)
	}

	return MessageOptions{
		Text:        res.Value,
		PhotoFileID: res.Image,
		VideoFileID: res.Video,
	}
}

// getText -отримання локалізації по мові та ключу
func (b *Bot) getText(key, languageCode string) (options string) {
	var res models.Localization

	b.localMutex.Lock()
	localizationMap, ok := b.localizations[languageCode]
	if ok {
		res = localizationMap[key]
	}
	b.localMutex.Unlock()

	if res.Value != "" {
		return res.Value
	}

	// Значення не знайдено, пробуємо отримати з бази
	logger.Info.Println("Key not found: ", key, languageCode)
	return b.service.GetText(key, languageCode)
}

// SendMessage отправляет новое сообщение с указанными опциями
func (b *Bot) SendMessage(chatID int64, options MessageOptions) error {
	// Определяем тип сообщения
	if options.MethodName == "" {
		if options.PhotoPath != "" || options.PhotoFileID != "" {
			options.MethodName = sendPhoto
		} else if options.VideoPath != "" || options.VideoFileID != "" {
			options.MethodName = sendVideo
		} else {
			options.MethodName = sendMessage
		}
	}

	// Устанавливаем в очередь на отправку
	return b.MakeRequestDeferred(chatID, 0, options)
}

// UpdateMessage обновляет существующее сообщение с указанными опциями
func (b *Bot) UpdateMessage(chatID int64, messageID int, options MessageOptions) error {
	// Если указан путь к фото
	if options.PhotoPath != "" {
		// Для фото с локального источника необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else if options.PhotoFileID != "" {
		// Обновление фото по FileID
		options.MethodName = editMessageMedia
		options.MessageID = messageID
		// Устанавливаем в очередь на отправку
		return b.MakeRequestDeferred(chatID, 0, options)

	} else if options.ReplyKeyboard != nil || options.RemoveKeyboard {
		// Для ReplyKeyboard необходимо удалить старое сообщение и отправить новое
		err := b.bot.DeleteMessage(b.ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}

		return b.SendMessage(chatID, options)
	} else {
		options.MethodName = editMessageText
		options.MessageID = messageID

		// Устанавливаем в очередь на отправку
		return b.MakeRequestDeferred(chatID, 0, options)
	}
}

// sendText отправляет текстовое сообщение
func (b *Bot) sendText(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      options.Text,
		ParseMode: options.ParseMode,
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: options.DisableWebPagePreview,
		},
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	msg, err := b.bot.SendMessage(b.ctx, params)
	if err != nil {
		logger.Error.Printf("Error sending message: %v", err)
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return msg, nil
}

// updateText обновляет текстовое сообщение
func (b *Bot) updateText(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      options.Text,
		ParseMode: options.ParseMode,
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: options.DisableWebPagePreview,
		},
	}

	// Добавляем сущности, если они есть
	if len(options.Entities) > 0 {
		params.Entities = options.Entities
	}

	// Для обновления можно использовать только инлайн клавиатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageText(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return msg, nil
}

// sendVideo отправляет видео с подписью
func (b *Bot) sendVideo(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Параметры для отправки
	params := &telego.SendVideoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		params.CaptionEntities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Выбираем источник видео
	if options.VideoFileID != "" {
		// Для FileID используем его непосредственно
		// params.Video = telego.InputFile{FileID: options.VideoFileID}
		params.Video = tu.FileFromID(options.VideoFileID)
		return b.bot.SendVideo(b.ctx, params)
	} else if options.VideoPath != "" {
		// Для файла используем метод Upload
		return b.sendVideoFile(chatID, options.VideoPath, params)
	}

	return nil, fmt.Errorf("no video source specified")
}

// sendPhoto отправляет фото с подписью
func (b *Bot) sendPhoto(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Параметры для отправки
	params := &telego.SendPhotoParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		params.CaptionEntities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Выбираем источник фото
	if options.PhotoFileID != "" {
		// Для FileID используем его непосредственно
		// params.Photo = telego.InputFile{FileID: options.PhotoFileID}
		params.Photo = tu.FileFromID(options.PhotoFileID)
		return b.bot.SendPhoto(b.ctx, params)
	} else if options.PhotoPath != "" {
		// Для файла используем метод Upload
		return b.sendPhotoFile(chatID, options.PhotoPath, options.DelPhoto, params)
	}

	return nil, fmt.Errorf("no photo source specified")
}

// sendAnimation отправляет анимацию с подписью
func (b *Bot) sendAnimation(chatID int64, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Параметры для отправки
	params := &telego.SendAnimationParams{
		ChatID:              telego.ChatID{ID: chatID},
		Caption:             options.Text,
		ParseMode:           options.ParseMode,
		DisableNotification: options.DisableNotification,
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		params.CaptionEntities = options.Entities
	}

	// Устанавливаем соответствующую клавиатуру
	if replyMarkup := b.getReplyMarkup(options); replyMarkup != nil {
		params.ReplyMarkup = replyMarkup
	}

	// Выбираем источник фото
	if options.VideoFileID != "" {
		// Для FileID используем его непосредственно
		params.Animation = tu.FileFromID(options.VideoFileID)
		return b.bot.SendAnimation(b.ctx, params)
	} else if options.VideoPath != "" {
		// Для файла используем метод Upload
		return b.sendAnimationFile(chatID, options.VideoPath, params)
	}

	return nil, fmt.Errorf("no animation source specified")
}

// sendAnimationFile отправляет видео с локального файла
func (b *Bot) sendAnimationFile(chatID int64, videoPath string, params *telego.SendAnimationParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(sharedData + "/video/" + videoPath + ".mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to open animation %s: %w", videoPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error.Println(err)
		}
	}()
	params.Animation = tu.File(file)

	// Отправляем анимацию
	msg, err := b.bot.SendAnimation(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send animation: %w", err)
	}

	if msg.Animation != nil {
		logger.Error.Println("videoPath", videoPath, msg.Animation.FileID)

		cont, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// записуємо в список анімацій
		err = b.redisDB.HSet(cont, gameAnimationPrefix, videoPath, msg.Animation.FileID).Err()
		if err != nil {
			logger.Error.Printf("Error Set %s: %v", videoPath, err)
		}
	}

	return msg, nil
}

// sendVideoFile отправляет видео с локального файла
func (b *Bot) sendVideoFile(chatID int64, videoPath string, params *telego.SendVideoParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error.Println(err)
		}
	}()
	params.Video = tu.File(file)

	// Отправляем фото
	msg, err := b.bot.SendVideo(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send video: %w", err)
	}

	return msg, nil
}

// sendPhotoFile отправляет фото с локального файла
func (b *Bot) sendPhotoFile(chatID int64, photoPath string, delPhoto bool, params *telego.SendPhotoParams) (*telego.Message, error) {
	// Открываем файл
	file, err := os.Open(photoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open photo file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error.Println(err)
		}
	}()

	if delPhoto {
		// Если был передан параметр на удаление
		// то через какое то время запускаем функцию удаления фото
		go func() {
			time.Sleep(20 * time.Second)
			if e := os.Remove(photoPath); e != nil {
				logger.Error.Println("failed to remove photo file: %w", err)
			}
		}()
	}

	// Устанавливаем загруженный файл
	params.Photo = tu.File(file)

	// Отправляем фото
	msg, err := b.bot.SendPhoto(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send photo: %w", err)
	}

	return msg, nil
}

// updatePhoto обновляет фото
func (b *Bot) updatePhoto(chatID int64, messageID int, options MessageOptions) (*telego.Message, error) {
	if options.ParseMode == "" {
		options.ParseMode = telego.ModeHTML
	}

	// Создаем объект InputMediaPhoto с FileID
	mediaPhoto := &telego.InputMediaPhoto{
		Type:      "photo",
		Caption:   options.Text,
		ParseMode: options.ParseMode,
	}

	// Выбираем источник фото
	if options.PhotoFileID != "" {
		mediaPhoto.Media = tu.FileFromID(options.PhotoFileID)
	} else if options.PhotoPath != "" {
		// Открываем файл
		file, err := os.Open(options.PhotoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open photo file: %w", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				logger.Error.Println(err)
			}
		}()

		if options.DelPhoto {
			// Если был передан параметр на удаление
			// то через какое то время запускаем функцию удаления фото
			go func() {
				time.Sleep(20 * time.Second)
				if e := os.Remove(options.PhotoPath); e != nil {
					logger.Error.Println("failed to remove photo file: %w", err)
				}
			}()
		}

		// Устанавливаем загруженный файл
		mediaPhoto.Media = tu.File(file)
	} else {
		return nil, fmt.Errorf("no photo source specified")
	}

	// Добавляем сущности для подписи, если они есть
	if len(options.Entities) > 0 {
		mediaPhoto.CaptionEntities = options.Entities
	}

	params := &telego.EditMessageMediaParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Media:     mediaPhoto,
	}

	// Для обновления можно использовать только инлайн клавиатуру
	if options.InlineKeyboard != nil {
		params.ReplyMarkup = options.InlineKeyboard
	}

	msg, err := b.bot.EditMessageMedia(b.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}

	return msg, nil
}

// getReplyMarkup возвращает соответствующую клавиатуру на основе опций
func (b *Bot) getReplyMarkup(options MessageOptions) telego.ReplyMarkup {
	if options.InlineKeyboard != nil {
		return options.InlineKeyboard
	}

	if options.RemoveKeyboard {
		return &telego.ReplyKeyboardRemove{
			RemoveKeyboard: true,
			Selective:      options.Selective,
		}
	}

	if options.ReplyKeyboard != nil {
		// Клонируем клавиатуру, чтобы не изменять оригинал
		keyboard := *options.ReplyKeyboard
		keyboard.OneTimeKeyboard = options.OneTimeKeyboard
		keyboard.Selective = options.Selective
		return &keyboard
	}

	return nil
}

// SendSticker отправляет стикер
func (b *Bot) SendSticker(chatID int64, stickerFileID string) error {
	_, err := b.bot.SendSticker(b.ctx, &telego.SendStickerParams{
		ChatID:  telego.ChatID{ID: chatID},
		Sticker: tu.FileFromID(stickerFileID),
	})
	return err
}

// // Случайный выбор стикера из двух вариантов
// func getRandomSticker(sticker1, sticker2 string) string {
// 	if time.Now().UnixNano()%2 == 0 {
// 		return sticker1
// 	}
// 	return sticker2
// }
