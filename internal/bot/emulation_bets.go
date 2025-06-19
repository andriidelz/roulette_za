package bot

import (
	"log"
	"roulette/internal/models"
	"time"
)

type GameDebugTask struct {
	debug        bool
	telegramID   int64 // telegramID под которым выполняется задание
	createdCount int   // кол-во созданных нажатий по заданию (около 3 на ставку)
	taskCount    int   // кол-во необходимых нажатий по заданию (около 3 на ставку)
}

var gameDebugTaskMap = map[int64]GameDebugTask{
	291591440: {
		debug:        true,
		telegramID:   291591440,
		createdCount: 0,
		taskCount:    10000,
	},
	1108851809: {
		debug:        true,
		telegramID:   1108851809,
		createdCount: 0,
		taskCount:    10000,
	}}

func (h *GameHandler) initEmulate() {
	go func() {
		// Периодический запуск ставок
		betTicker := time.NewTicker(5 * time.Second)
		defer betTicker.Stop()

		for {
			select {
			case tm := <-betTicker.C:

				// Для всех пользователей которые имеют задания
				for telegramID := range gameDebugTaskMap {
					h.gameHandlerEmulate(telegramID, tm.Unix())
				}
			}
		}
	}()
}

// gameHandlerEmulate эмулирует нажатие кнопки сделать ставку
func (h *GameHandler) gameHandlerEmulate(telegramID, remainingSeconds int64) {
	// Проверяем вошел ли пользователь в игру
	if ok := h.activePlayers[telegramID]; !ok {
		return
	}

	// Получаем задание
	task, ok := gameDebugTaskMap[telegramID]
	if !ok || !task.debug {
		return
	}
	// Если кол-во нажатий выполнено то остановка игры
	if task.taskCount <= task.createdCount {
		log.Println("Total emulate bets: ", task.createdCount, telegramID)

		h.HandleStopGameButton(telegramID)
		return
	}

	// Обновляем счетчик
	task.createdCount++
	if task.createdCount%100 == 0 {
		log.Println("emulate bet: ", task.createdCount, telegramID)
	}
	gameDebugTaskMap[telegramID] = task

	//  Если для пользователя есть эмуляция ставки то имитируем нажатие на кнопку красное/черное
	bet := models.Red
	if remainingSeconds%2 == 0 {
		bet = models.Black
	}
	h.bot.handleMakeBet(task.telegramID, bet)
}
