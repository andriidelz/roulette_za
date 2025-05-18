package bot

import (
	"sync"
)

// UserState представляет состояние пользователя в боте
type UserState struct {
	State     string         // Текущее состояние
	Data      map[string]any // Дополнительные данные
	MessageID int            // ID сообщения для обновления
}

// StateManager управляет состояниями пользователей
type StateManager struct {
	states map[int64]*UserState
	mu     sync.RWMutex
}

// Константы состояний
const (
	StateNone            = ""
	StateInputName       = "input_name"
	StateInputWallet     = "input_wallet"
	StateInputNickname   = "input_nickname"        // создание никнейма при регистрации
	StateInputUpNickname = "input_update_nickname" // обновление никнейма в настройках
)

// NewStateManager создает новый экземпляр менеджера состояний
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[int64]*UserState),
	}
}

// SetState устанавливает состояние для пользователя
func (sm *StateManager) SetState(userID int64, state string, messageID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[userID]; !exists {
		sm.states[userID] = &UserState{
			Data: make(map[string]any),
		}
	}

	sm.states[userID].State = state
	sm.states[userID].MessageID = messageID
}

// GetState получает текущее состояние пользователя
func (sm *StateManager) GetState(userID int64) (string, int, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[userID]; exists {
		return state.State, state.MessageID, true
	}

	return StateNone, 0, false
}

// ClearState очищает состояние пользователя
func (sm *StateManager) ClearState(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, userID)
}
