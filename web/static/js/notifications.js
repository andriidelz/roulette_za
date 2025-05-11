/**
 * Сохранение шаблона уведомления
 */
function saveTemplate() {
    // Проверяем, идет ли сохранение в данный момент
    if (document.getElementById('saving-indicator')) {
        console.log('Save already in progress, ignoring duplicate request');
        return;
    }
    
    // Создаем индикатор сохранения
    const saveButton = document.querySelector('#editTemplateModal .modal-footer button[type="submit"]');
    if (saveButton) {
        saveButton.disabled = true;
        saveButton.innerHTML = '<span id="saving-indicator" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Сохранение...';
    }
    
    try {
        // Собираем данные формы
        const id = document.getElementById('template-id').value;
        const name = document.getElementById('template-name').value;
        const type = document.getElementById('template-type').value || document.getElementById('hidden-template-type')?.value;
        const triggerEvent = type === 'automatic' ? document.getElementById('trigger-event').value : '';
        
        // Получаем ключи локализации
        const titleKey = document.getElementById('title-key').value;
        const messageKey = document.getElementById('message-key').value;
        
        // Собираем локализации заголовка
        const title = {
            en: document.getElementById('title-en').value,
            ru: document.getElementById('title-ru').value,
            uk: document.getElementById('title-uk').value
        };
        
        // Собираем локализации сообщения
        const message = {
            en: document.getElementById('message-en').value,
            ru: document.getElementById('message-ru').value,
            uk: document.getElementById('message-uk').value
        };
        
        // Собираем URL изображений для разных языков
        const image = {};
        const enImage = document.getElementById('edit-image-en-input').value;
        const ruImage = document.getElementById('edit-image-ru-input').value;
        const ukImage = document.getElementById('edit-image-uk-input').value;
        
        // Только если поля не пусты, добавляем их в объект
        if (enImage) image.en = enImage;
        if (ruImage) image.ru = ruImage;
        if (ukImage) image.uk = ukImage;
        
        // Проверяем обязательные поля
        if (!name) {
            showNotification('Пожалуйста, введите название шаблона', 'danger');
            resetSaveButton();
            return;
        }
        
        if (!titleKey) {
            showNotification('Пожалуйста, введите ключ локализации для заголовка', 'danger');
            resetSaveButton();
            return;
        }
        
        if (!messageKey) {
            showNotification('Пожалуйста, введите ключ локализации для сообщения', 'danger');
            resetSaveButton();
            return;
        }

        const active = document.getElementById('template-active').checked;
        
        // Подготовка данных для отправки
        const data = {
            name: name,
            type: type,
            trigger_event: triggerEvent,
            title_key: titleKey,
            message_key: messageKey,
            title: title,
            message: message,
            image: image,
            active: active,
            button: {}
        };
        
        // Добавляем настройки кнопки, если включены
        if (document.getElementById('has-button').checked) {
            const buttonTextKey = document.getElementById('button-text-key').value;
            const buttonUrl = document.getElementById('edit-button-url').value;
            const buttonCallback = document.getElementById('edit-button-callback').value;
            
            // Собираем локализации кнопки
            const buttonText = {
                en: document.getElementById('button-text-en').value || '',
                ru: document.getElementById('button-text-ru').value || '',
                uk: document.getElementById('button-text-uk').value || ''
            };
            
            data.button = {
                text_key: buttonTextKey,
                text: buttonText,
                url: buttonUrl,
                callback: buttonCallback
            };
        }
        
        console.log('Отправляемые данные:', data);
        
        // Определяем URL и метод запроса
        const requestUrl = id ? `/admin/api/notification-templates/${id}` : '/admin/api/notification-templates';
        const method = id ? 'PUT' : 'POST';
        
        // Отправляем запрос на сервер
        fetch(requestUrl, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        })
        .then(response => {
            if (!response.ok) {
                return response.json().then(err => Promise.reject(err));
            }
            return response.json();
        })
        .then(result => {
            console.log('Ответ сервера:', result);
            
            // Закрываем модальное окно
            const modal = bootstrap.Modal.getInstance(document.getElementById('editTemplateModal'));
            modal.hide();
            
            // Показываем сообщение об успехе
            showNotification(id ? 'Шаблон успешно обновлен' : 'Шаблон успешно создан', 'success');
            
            // Обновляем страницу
            setTimeout(() => {
                window.location.reload();
            }, 1000);
        })
        .catch(error => {
            console.error('Error saving template:', error);
            showNotification(error.error || 'Ошибка сохранения шаблона', 'danger');
        })
        .finally(() => {
            resetSaveButton();
        });
    } catch (error) {
        console.error('Unexpected error:', error);
        showNotification('Произошла неожиданная ошибка: ' + error.message, 'danger');
        resetSaveButton();
    }
    
    // Функция для сброса кнопки сохранения
    function resetSaveButton() {
        const saveButton = document.querySelector('#editTemplateModal .modal-footer button[type="submit"]');
        if (saveButton) {
            saveButton.disabled = false;
            saveButton.innerHTML = 'Сохранить';
        }
    }
}

/**
 * Функция для отображения уведомлений в стиле Bootstrap
 * @param {string} message Текст сообщения
 * @param {string} type Тип сообщения (success, info, warning, danger)
 */
function showNotification(message, type) {
    // Создаем контейнер для уведомлений, если его нет
    let container = document.getElementById('notification-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'notification-container';
        container.className = 'position-fixed top-0 end-0 p-3';
        container.style.zIndex = '1050';
        document.body.appendChild(container);
    }
    
    // Создаем элемент уведомления
    const notificationId = 'notification-' + Date.now();
    const notification = document.createElement('div');
    notification.id = notificationId;
    notification.className = `toast align-items-center text-white bg-${type} border-0`;
    notification.setAttribute('role', 'alert');
    notification.setAttribute('aria-live', 'assertive');
    notification.setAttribute('aria-atomic', 'true');
    
    // Добавляем содержимое уведомления
    notification.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">
                ${message}
            </div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"></button>
        </div>
    `;
    
    // Добавляем уведомление в контейнер
    container.appendChild(notification);
    
    // Инициализируем и показываем уведомление с помощью Bootstrap
    const toast = new bootstrap.Toast(notification, {
        delay: 5000
    });
    toast.show();
    
    // Удаляем уведомление после скрытия
    notification.addEventListener('hidden.bs.toast', function() {
        notification.remove();
    });
}

/**
 * Удаление шаблона уведомления
 * @param {string} id ID шаблона
 */
function deleteTemplate(id) {
    // Отправляем запрос на удаление
    fetch(`/admin/api/notification-templates/${id}`, {
        method: 'DELETE'
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => Promise.reject(err));
        }
        return response.json();
    })
    .then(result => {
        toastr.success('Шаблон успешно удален');
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    })
    .catch(error => {
        console.error('Error deleting template:', error);
        toastr.error(error.error || 'Ошибка удаления шаблона');
    });
}

/**
 * Создание задачи на отправку уведомления
 * @param {string} templateId ID шаблона
 * @param {string} templateName Название шаблона
 */
function createTask(templateId, templateName) {
    // Устанавливаем ID шаблона в форме задачи
    document.getElementById('task-template-id').value = templateId;
    document.getElementById('task-template-name').textContent = templateName;
    
    // Сбрасываем форму
    document.getElementById('taskForm').reset();
    
    // Устанавливаем дату и время по умолчанию (через 15 минут от текущего времени)
    const now = new Date();
    now.setMinutes(now.getMinutes() + 15);
    const dateTimeString = now.toISOString().slice(0, 16);
    document.getElementById('scheduled-at').value = dateTimeString;
    
    // Скрываем контейнеры таргетинга и настройки расписания
    document.getElementById('target-countries-container').style.display = 'none';
    document.getElementById('target-activity-container').style.display = 'none';
    document.getElementById('scheduled-settings').style.display = 'none';
    document.getElementById('adjust-time-settings').style.display = 'none';
    
    // Загружаем данные для фильтров
    loadCountriesForFilter();
    
    // Открываем модальное окно
    const modal = new bootstrap.Modal(document.getElementById('createTaskModal'));
    modal.show();
}

/**
 * Загрузка списка стран для фильтра
 */
function loadCountriesForFilter() {
    const countriesList = document.getElementById('countries-list');
    
    // Показываем индикатор загрузки
    countriesList.innerHTML = '<div class="text-center py-3"><div class="spinner-border" role="status"><span class="visually-hidden">Загрузка...</span></div></div>';
    
    // Загружаем список стран
    fetch('/admin/api/countries-with-users')
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Очищаем список
            countriesList.innerHTML = '';
            
            // Добавляем опцию "Все страны"
            const allCountriesDiv = document.createElement('div');
            allCountriesDiv.className = 'mb-2';
            allCountriesDiv.innerHTML = `
                <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="country-all" value="all" checked>
                    <label class="form-check-label" for="country-all">
                        🌎 Все страны
                    </label>
                </div>
            `;
            countriesList.appendChild(allCountriesDiv);
            
            // Сортируем страны по количеству пользователей
            data.sort((a, b) => b.count - a.count);
            
            // Добавляем все страны в список
            data.forEach(country => {
                const countryDiv = document.createElement('div');
                countryDiv.className = 'mb-2';
                countryDiv.innerHTML = `
                    <div class="form-check">
                        <input class="form-check-input country-checkbox" type="checkbox" id="country-${country.code}" value="${country.code}" data-users="${country.count}" disabled>
                        <label class="form-check-label" for="country-${country.code}">
                            ${country.emoji || ''} ${country.name} (${country.count} пользователей)
                        </label>
                    </div>
                `;
                countriesList.appendChild(countryDiv);
            });
            
            // Добавляем обработчик для чекбокса "Все страны"
            document.getElementById('country-all').addEventListener('change', function() {
                const checked = this.checked;
                document.querySelectorAll('.country-checkbox').forEach(checkbox => {
                    checkbox.checked = checked;
                    checkbox.disabled = checked;
                });
            });
        })
        .catch(error => {
            console.error('Error loading countries:', error);
            countriesList.innerHTML = `<div class="alert alert-danger">
                Ошибка загрузки списка стран: ${error.message}
            </div>`;
        });
}

/**
 * Создание задачи на отправку уведомления
 */
function createNotificationTask() {
    // Собираем данные формы
    const templateId = document.getElementById('task-template-id').value;
    const targetType = document.getElementById('task-target-type').value;
    
    // Собираем параметры таргетинга в зависимости от типа
    let targetParams = {};
    
    if (targetType === 'country') {
        // Если выбран чекбокс "Все страны", параметры не нужны
        if (document.getElementById('country-all').checked) {
            targetParams = { allCountries: true };
        } else {
            // Собираем выбранные страны
            const selectedCountries = [];
            document.querySelectorAll('.country-checkbox:checked').forEach(checkbox => {
                selectedCountries.push(checkbox.value);
            });
            
            if (selectedCountries.length === 0) {
                toastr.error('Пожалуйста, выберите хотя бы одну страну');
                return;
            }
            
            targetParams = { countries: selectedCountries };
        }
    } else if (targetType === 'activity') {
        // Получаем выбранный фильтр активности из select
        const activityFilter = document.getElementById('activity-filter-select').value;
        
        if (!activityFilter) {
            toastr.error('Пожалуйста, выберите фильтр активности');
            return;
        }
        
        targetParams = { activityFilters: [activityFilter] };
    }
    
    // Проверяем, включена ли отложенная отправка
    const scheduledSend = document.getElementById('scheduled-send').checked;
    let scheduledAt = null;
    let adjustTime = false;
    
    if (scheduledSend) {
        scheduledAt = document.getElementById('scheduled-at').value;
        
        if (!scheduledAt) {
            toastr.error('Пожалуйста, укажите дату и время отправки');
            return;
        }
        
        // Проверяем, включена ли адаптация времени
        adjustTime = document.getElementById('adjust-time').checked;
        
        if (adjustTime) {
            const sendTimeStart = document.getElementById('send-time-start').value;
            const sendTimeEnd = document.getElementById('send-time-end').value;
            
            if (!sendTimeStart || !sendTimeEnd) {
                toastr.error('Пожалуйста, укажите временной интервал для адаптации времени');
                return;
            }
            
            targetParams.sendTimeStart = sendTimeStart;
            targetParams.sendTimeEnd = sendTimeEnd;
        }
    }
    
    // Подготовка данных для отправки
    const data = {
        templateId: parseInt(templateId),
        targetType,
        targetParams,
        scheduledSend,
        scheduledAt: scheduledSend ? scheduledAt : undefined,
        adjustTime
    };
    
    // Отправляем запрос на сервер
    fetch('/admin/api/notification-tasks', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => Promise.reject(err));
        }
        return response.json();
    })
    .then(result => {
        // Закрываем модальное окно
        const modal = bootstrap.Modal.getInstance(document.getElementById('createTaskModal'));
        modal.hide();
        
        // Показываем сообщение об успехе
        toastr.success(`Задача на отправку уведомления создана успешно. Всего получателей: ${result.totalUsers}`);
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1500);
    })
    .catch(error => {
        console.error('Error creating task:', error);
        toastr.error(error.error || 'Ошибка создания задачи');
    });
}

/**
 * Просмотр задачи на отправку уведомления
 * @param {string} id ID задачи
 */
function viewTask(id) {
    // Показываем лоадер
    document.getElementById('viewTaskLoader').style.display = 'block';
    document.getElementById('viewTaskContent').style.display = 'none';
    document.getElementById('cancel-task-btn').style.display = 'none';
    
    // Открываем модальное окно
    const viewTaskModal = new bootstrap.Modal(document.getElementById('viewTaskModal'));
    viewTaskModal.show();
    
    // Загружаем данные задачи
    fetch(`/admin/api/notification-tasks/${id}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Заполняем общую информацию
            const templateName = data.templateWithLocalizations ? 
            (data.templateWithLocalizations.name || 'Неизвестный шаблон') : 
            'Неизвестный шаблон';
            
            document.getElementById('view-task-template-name').textContent = templateName;
            
            // Определяем статус и соответствующий класс
            let statusText = '';
            let statusClass = '';
            switch (data.status) {
                case 'pending':
                    statusText = 'Ожидает';
                    statusClass = 'bg-warning';
                    // Показываем кнопку отмены для ожидающих задач
                    document.getElementById('cancel-task-btn').style.display = 'block';
                    document.getElementById('cancel-task-btn').setAttribute('data-id', id);
                    break;
                case 'processing':
                    statusText = 'Выполняется';
                    statusClass = 'bg-primary';
                    // Показываем кнопку отмены для выполняющихся задач
                    document.getElementById('cancel-task-btn').style.display = 'block';
                    document.getElementById('cancel-task-btn').setAttribute('data-id', id);
                    break;
                case 'completed':
                    statusText = 'Завершено';
                    statusClass = 'bg-success';
                    break;
                case 'failed':
                    statusText = 'Ошибка';
                    statusClass = 'bg-danger';
                    break;
                case 'canceled':
                    statusText = 'Отменено';
                    statusClass = 'bg-secondary';
                    break;
            }
            document.getElementById('view-task-status').innerHTML = `<span class="badge ${statusClass}">${statusText}</span>`;
            
            // Определяем и отображаем тип таргетинга
            let targetTypeText = '';
            switch (data.targetType) {
                case 'all':
                    targetTypeText = 'Все пользователи';
                    break;
                case 'country':
                    targetTypeText = 'По странам';
                    break;
                case 'activity':
                    targetTypeText = 'По активности';
                    break;
            }
            document.getElementById('view-task-target-type').textContent = targetTypeText;
            
            // Отображаем запланированное время
            if (data.scheduledAt) {
                const scheduledDate = new Date(data.scheduledAt);
                document.getElementById('view-task-scheduled-at').textContent = scheduledDate.toLocaleString();
            } else {
                document.getElementById('view-task-scheduled-at').textContent = 'Немедленно';
            }
            
            // Отображаем прогресс
            const progress = typeof data.taskProgress !== 'undefined' ? data.taskProgress : 0;
            const progressBar = document.getElementById('view-task-progress-bar');
            progressBar.style.width = `${progress.toFixed(1)}%`;
            progressBar.textContent = `${progress.toFixed(1)}%`;
            progressBar.setAttribute('aria-valuenow', progress);
            
            // Отображаем статистику
            document.getElementById('view-task-total-users').textContent = data.totalUsers;
            document.getElementById('view-task-sent-count').textContent = data.sentCount;
            document.getElementById('view-task-delivered-count').textContent = data.deliveredCount;
            document.getElementById('view-task-read-count').textContent = data.readCount;
            
            // Отображаем оставшееся время (если задача в процессе)
            const timeRemainingContainer = document.getElementById('view-task-time-remaining-container');
            if (data.status === 'processing' && data.estimatedTimeRemaining) {
                timeRemainingContainer.style.display = 'block';
                document.getElementById('view-task-time-remaining').textContent = data.estimatedTimeRemaining;
            } else {
                timeRemainingContainer.style.display = 'none';
            }
            
            // Отображаем параметры таргетинга
            let targetParamsHTML = '<p>Все пользователи</p>';
            if (data.targetType === 'country' && data.targetParams && data.targetParams.countries) {
                targetParamsHTML = '<p><strong>Выбранные страны:</strong></p><ul>';
                data.targetParams.countries.forEach(country => {
                    targetParamsHTML += `<li>${country}</li>`;
                });
                targetParamsHTML += '</ul>';
            } else if (data.targetType === 'activity' && data.targetParams && data.targetParams.activityFilters) {
                // Поскольку теперь у нас только один фильтр активности, упростим отображение
                const filter = data.targetParams.activityFilters[0];
                let filterText = '';
                
                switch (filter) {
                    case 'inactive_3days':
                        filterText = 'Не играл менее 3 дней (от 3 дней до 12 часов)';
                        break;
                    case 'inactive_7days':
                        filterText = 'Не играл более 3 дней и менее 7 дней';
                        break;
                    case 'inactive_14days':
                        filterText = 'Не играл более 7 дней и менее 14 дней';
                        break;
                    case 'inactive_more_14days':
                        filterText = 'Не играл более 14 дней';
                        break;
                    default:
                        filterText = filter;
                }
                
                targetParamsHTML = `<p><strong>Фильтр активности:</strong> ${filterText}</p>`;
            }
            document.getElementById('view-task-target-params').innerHTML = targetParamsHTML;
            
            // Отображаем время выполнения
            document.getElementById('view-task-created-at').textContent = data.createdAt ? new Date(data.createdAt).toLocaleString() : '-';
            document.getElementById('view-task-started-at').textContent = data.startedAt ? new Date(data.startedAt).toLocaleString() : '-';
            document.getElementById('view-task-completed-at').textContent = data.completedAt ? new Date(data.completedAt).toLocaleString() : '-';
            
            // Скрываем лоадер и показываем контент
            document.getElementById('viewTaskLoader').style.display = 'none';
            document.getElementById('viewTaskContent').style.display = 'block';
        })
        .catch(error => {
            console.error('Error loading task:', error);
            // Показываем ошибку в лоадере
            document.getElementById('viewTaskLoader').innerHTML = `
            <div class="alert alert-danger">
                Ошибка загрузки данных: ${error.message}
                <button type="button" class="btn-close float-end" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
            `;
        });
}

/**
 * Отмена задачи на отправку уведомления
 * @param {string} id ID задачи
 */
function cancelTask(id) {
    fetch(`/admin/api/notification-tasks/${id}/cancel`, {
        method: 'POST'
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => Promise.reject(err));
        }
        return response.json();
    })
    .then(result => {
        // Если было открыто модальное окно, закрываем его
        const viewTaskModal = bootstrap.Modal.getInstance(document.getElementById('viewTaskModal'));
        if (viewTaskModal) {
            viewTaskModal.hide();
        }
        
        // Показываем сообщение об успехе
        toastr.success('Задача успешно отменена');
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    })
    .catch(error => {
        console.error('Error canceling task:', error);
        toastr.error(error.error || 'Ошибка отмены задачи');
    });
}

/**
 * Поиск и фильтрация шаблонов
 */
function searchTemplates() {
    const query = document.getElementById('search-templates').value.toLowerCase();
    filterTemplates(query);
}

/**
 * Фильтрация шаблонов по различным параметрам
 * @param {string} query Строка поиска
 */
function filterTemplates(query = '') {
  // Получаем параметры фильтрации
  const showOnlyActive = document.getElementById('show-only-active')?.checked || false;
  
  // Проверяем наличие фильтра типа шаблона
  let templateType = 'all';
  const templateTypeFilter = document.getElementById('template-type-filter');
  if (templateTypeFilter) {
    templateType = templateTypeFilter.value || 'all';
  }
  
  // Применяем фильтры ко всем строкам таблицы
  const rows = document.querySelectorAll('#templates-list tr');
  rows.forEach(row => {
    // Пропускаем заголовок и строки без данных
    if (!row.querySelector('td')) return;
    
    // Получаем ячейки строки
    const nameCell = row.querySelector('td:nth-child(1)');
    const typeCell = row.querySelector('td:nth-child(2)');
    const statusCell = row.querySelector('td:nth-child(4)');
    
    if (!nameCell || !typeCell || !statusCell) return;
    
    // Проверяем соответствие поисковому запросу
    const name = nameCell.textContent.toLowerCase();
    const matches = !query || name.includes(query);
    
    // Проверяем соответствие фильтру типа
    const type = typeCell.textContent.toLowerCase();
    const matchesType = templateType === 'all' || 
       (templateType === 'manual' && type.includes('ручной')) ||
       (templateType === 'automatic' && type.includes('автоматический'));
    
    // Проверяем соответствие фильтру активности
    const isActive = statusCell.querySelector('.badge.bg-success') !== null;
    const matchesActive = !showOnlyActive || isActive;
    
    // Показываем или скрываем строку
    row.style.display = matches && matchesType && matchesActive ? '' : 'none';
  });
}

/**
 * Файл для работы с уведомлениями в админ-панели
 */

document.addEventListener('DOMContentLoaded', function() {
    // Инициализация уведомлений, если мы находимся на странице уведомлений
    // и это еще не было сделано
    if (document.querySelector('.notifications-page') && !window.notificationsInitialized) {
        window.notificationsInitialized = true;
        initNotifications();
    }
});

/**
 * Инициализация компонентов и обработчиков на странице уведомлений
 */
function initNotifications() {
    // Инициализация шаблонов
    initializeTemplateHandlers();
    
    // Инициализация форм
    initializeNotificationForms();
    
    // Инициализация диаграмм для статистики
    if (document.getElementById('notifications-chart')) {
        initializeNotificationCharts();
    }
    
    // Фильтрация автоматических шаблонов
    if (document.getElementById('show-only-active-auto')) {
        document.getElementById('show-only-active-auto').addEventListener('change', function() {
            const showOnlyActive = this.checked;
            const rows = document.querySelectorAll('#automatic-templates-list tr.template-row');
            
            rows.forEach(row => {
                const isActive = row.querySelector('.badge.bg-success') !== null;
                
                if (showOnlyActive && !isActive) {
                    row.style.display = 'none';
                } else {
                    row.style.display = '';
                }
            });
        });
        
        // При загрузке применяем фильтр
        document.getElementById('show-only-active-auto').dispatchEvent(new Event('change'));
    }
    
    // Применить фильтры для таблицы шаблонов при загрузке страницы
    if (document.getElementById('show-only-active')) {
        filterTemplates();
    }
}

// Глобальные флаги для предотвращения множественных запросов
window.processingTemplate = false;
window.processingTask = false;

/**
 * Инициализация обработчиков для шаблонов уведомлений
 */
function initializeTemplateHandlers() {
    // Сначала удалим все существующие обработчики, чтобы избежать дублирования
    document.querySelectorAll('.view-template').forEach(button => {
        button.removeEventListener('click', viewTemplateHandler);
        // Добавляем новый обработчик с trackable функцией
        button.addEventListener('click', viewTemplateHandler);
    });
    
    document.querySelectorAll('.edit-template').forEach(button => {
        button.removeEventListener('click', editTemplateHandler);
        button.addEventListener('click', editTemplateHandler);
    });
    
    document.querySelectorAll('.create-task').forEach(button => {
        button.removeEventListener('click', createTaskHandler);
        button.addEventListener('click', createTaskHandler);
    });
    
    document.querySelectorAll('.delete-template').forEach(button => {
        button.removeEventListener('click', deleteTemplateHandler);
        button.addEventListener('click', deleteTemplateHandler);
    });
    
    document.querySelectorAll('.view-task').forEach(button => {
        button.removeEventListener('click', viewTaskHandler);
        button.addEventListener('click', viewTaskHandler);
    });
    
    document.querySelectorAll('.cancel-task').forEach(button => {
        button.removeEventListener('click', cancelTaskHandler);
        button.addEventListener('click', cancelTaskHandler);
    });
    
    // Кнопка создания нового шаблона
    const createTemplateBtn = document.getElementById('create-template-btn');
    if (createTemplateBtn) {
        createTemplateBtn.removeEventListener('click', resetTemplateFormHandler);
        createTemplateBtn.addEventListener('click', resetTemplateFormHandler);
    }
    
    // Фильтры и поиск
    const searchTemplatesBtn = document.getElementById('search-templates-btn');
    if (searchTemplatesBtn) {
        searchTemplatesBtn.removeEventListener('click', searchTemplates);
        searchTemplatesBtn.addEventListener('click', searchTemplates);
    }
    
    const searchTemplatesInput = document.getElementById('search-templates');
    if (searchTemplatesInput) {
        searchTemplatesInput.removeEventListener('keyup', searchTemplatesKeyup);
        searchTemplatesInput.addEventListener('keyup', searchTemplatesKeyup);
    }
    
    const showOnlyActiveCheckbox = document.getElementById('show-only-active');
    if (showOnlyActiveCheckbox) {
        showOnlyActiveCheckbox.removeEventListener('change', filterTemplatesOnChange);
        showOnlyActiveCheckbox.addEventListener('change', filterTemplatesOnChange);
    }
    
    const templateTypeFilter = document.getElementById('template-type-filter');
    if (templateTypeFilter) {
        templateTypeFilter.removeEventListener('change', filterTemplatesOnChange);
        templateTypeFilter.addEventListener('change', filterTemplatesOnChange);
    }
}

// Обработчики событий для кнопок
function viewTemplateHandler() {
    // Защита от множественных вызовов
    if (window.processingTemplate) return;
    window.processingTemplate = true;
    
    const templateId = this.getAttribute('data-id');
    viewTemplate(templateId);
    
    // Сбросим флаг через небольшую задержку
    setTimeout(() => {
        window.processingTemplate = false;
    }, 500);
}

function editTemplateHandler() {
    if (window.processingTemplate) return;
    window.processingTemplate = true;
    
    const templateId = this.getAttribute('data-id');
    editTemplate(templateId);
    
    setTimeout(() => {
        window.processingTemplate = false;
    }, 500);
}

function createTaskHandler() {
    if (window.processingTemplate) return;
    window.processingTemplate = true;
    
    const templateId = this.getAttribute('data-id');
    const templateName = this.getAttribute('data-name');
    createTask(templateId, templateName);
    
    setTimeout(() => {
        window.processingTemplate = false;
    }, 500);
}

function deleteTemplateHandler() {
    if (window.processingTemplate) return;
    window.processingTemplate = true;
    
    const templateId = this.getAttribute('data-id');
    if (confirm('Вы уверены, что хотите удалить этот шаблон?')) {
        deleteTemplate(templateId);
    } else {
        window.processingTemplate = false;
    }
}

function viewTaskHandler() {
    if (window.processingTask) return;
    window.processingTask = true;
    
    const taskId = this.getAttribute('data-id');
    viewTask(taskId);
    
    setTimeout(() => {
        window.processingTask = false;
    }, 500);
}

function cancelTaskHandler() {
    if (window.processingTask) return;
    window.processingTask = true;
    
    const taskId = this.getAttribute('data-id');
    if (confirm('Вы уверены, что хотите отменить эту задачу?')) {
        cancelTask(taskId);
    } else {
        window.processingTask = false;
    }
}

function resetTemplateFormHandler() {
    resetTemplateForm();
}

function searchTemplatesKeyup(e) {
    if (e.key === 'Enter') {
        searchTemplates();
    }
}

function filterTemplatesOnChange() {
    filterTemplates();
}

/**
 * Инициализация форм для уведомлений
 */
function initializeNotificationForms() {
    // Инициализация формы шаблона
    const templateForm = document.getElementById('templateForm');
    if (templateForm) {
        // Удаляем существующие обработчики перед добавлением нового
        templateForm.removeEventListener('submit', saveTemplate);
        templateForm.addEventListener('submit', function(e) {
            e.preventDefault();
            saveTemplate();
        });
    }
    
    // Инициализация формы задачи на отправку
    const taskForm = document.getElementById('taskForm');
    if (taskForm) {
        // Переключение видимости настроек таргетинга стран
        const targetTypeSelect = document.getElementById('task-target-type');
        if (targetTypeSelect) {
            targetTypeSelect.addEventListener('change', function() {
                const countriesContainer = document.getElementById('target-countries-container');
                const activityContainer = document.getElementById('target-activity-container');
                
                if (countriesContainer) {
                    countriesContainer.style.display = this.value === 'country' ? 'block' : 'none';
                }
                
                if (activityContainer) {
                    activityContainer.style.display = this.value === 'activity' ? 'block' : 'none';
                }
            });
        }
        
        // Переключение видимости настроек отложенной отправки
        const scheduledSendCheckbox = document.getElementById('scheduled-send');
        if (scheduledSendCheckbox) {
            scheduledSendCheckbox.addEventListener('change', function() {
                const scheduledSettings = document.getElementById('scheduled-settings');
                if (scheduledSettings) {
                    scheduledSettings.style.display = this.checked ? 'block' : 'none';
                }
            });
        }
        
        // Переключение видимости настроек адаптации времени
        const adjustTimeCheckbox = document.getElementById('adjust-time');
        if (adjustTimeCheckbox) {
            adjustTimeCheckbox.addEventListener('change', function() {
                const adjustTimeSettings = document.getElementById('adjust-time-settings');
                if (adjustTimeSettings) {
                    adjustTimeSettings.style.display = this.checked ? 'block' : 'none';
                }
            });
        }
        
        // Кнопка создания задачи
        const createTaskBtn = document.getElementById('create-task-btn');
        if (createTaskBtn) {
            createTaskBtn.addEventListener('click', function() {
                createNotificationTask();
            });
        }
        
        // Поиск по странам
        const searchCountriesInput = document.getElementById('search-countries');
        if (searchCountriesInput) {
            searchCountriesInput.addEventListener('keyup', function() {
                const value = this.value.toLowerCase();
                const countryItems = document.querySelectorAll('#countries-list .form-check');
                
                countryItems.forEach(item => {
                    const text = item.textContent.toLowerCase();
                    item.style.display = text.includes(value) ? '' : 'none';
                });
            });
        }
    }
}

// Глобальные переменные для хранения экземпляров диаграмм
if (typeof window.notificationsLineChart === 'undefined') {
    window.notificationsLineChart = null;
}
if (typeof window.notificationTypesChart === 'undefined') {
    window.notificationTypesChart = null;
}
if (typeof window.countryDistributionChart === 'undefined') {
    window.countryDistributionChart = null;
}

/**
 * Инициализация диаграмм для статистики уведомлений
 */
function initializeNotificationCharts() {
    // Проверяем доступность библиотеки Chart
    if (typeof Chart === 'undefined') {
        console.error('Chart.js не загружен. Пожалуйста, добавьте библиотеку Chart.js на страницу.');
        
        // Показываем сообщение об ошибке в контейнерах диаграмм
        const errorMessage = '<div class="alert alert-danger">Не удалось загрузить диаграммы: библиотека Chart.js не найдена</div>';
        
        const chartContainers = [
            document.getElementById('notifications-chart'),
            document.getElementById('notification-types-chart'),
            document.getElementById('country-distribution-chart')
        ];
        
        chartContainers.forEach(container => {
            if (container) {
                container.innerHTML = errorMessage;
            }
        });
        
        return;
    }

    // Инициализируем переменные для хранения экземпляров диаграмм, если они не существуют
    if (!window.chartInstances) {
        window.chartInstances = {};
    }
    
    // Получаем контексты для диаграмм
    const notificationsChartElem = document.getElementById('notifications-chart');
    const notificationTypesChartElem = document.getElementById('notification-types-chart');
    const countryDistributionChartElem = document.getElementById('country-distribution-chart');
    
    // Уничтожаем существующие диаграммы перед созданием новых
    const chartIds = ['notificationsLineChart', 'notificationTypesChart', 'countryDistributionChart'];
    chartIds.forEach(id => {
        if (window.chartInstances[id]) {
            window.chartInstances[id].destroy();
            window.chartInstances[id] = null;
        }
    });
    
    if (notificationsChartElem) {
        // Пример данных для диаграммы по дням
        const days = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
        const sentData = [65, 78, 52, 91, 83, 56, 70];
        const deliveredData = [63, 75, 48, 88, 80, 53, 68];
        const readData = [38, 45, 30, 53, 47, 32, 41];
        
        window.chartInstances.notificationsLineChart = new Chart(notificationsChartElem.getContext('2d'), {
            type: 'line',
            data: {
                labels: days,
                datasets: [
                    {
                        label: 'Отправлено',
                        data: sentData,
                        backgroundColor: 'rgba(54, 162, 235, 0.2)',
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 2,
                        tension: 0.3
                    },
                    {
                        label: 'Доставлено',
                        data: deliveredData,
                        backgroundColor: 'rgba(75, 192, 192, 0.2)',
                        borderColor: 'rgba(75, 192, 192, 1)',
                        borderWidth: 2,
                        tension: 0.3
                    },
                    {
                        label: 'Прочитано',
                        data: readData,
                        backgroundColor: 'rgba(255, 205, 86, 0.2)',
                        borderColor: 'rgba(255, 205, 86, 1)',
                        borderWidth: 2,
                        tension: 0.3
                    }
                ]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: 'Активность уведомлений по дням недели'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }
    
    if (notificationTypesChartElem) {
        // Пример данных для диаграммы типов уведомлений
        const types = ['Ручные', 'Регистрация', 'Рейтинг', 'Выигрыш', 'Активность'];
        const typeData = [30, 15, 20, 25, 10];
        const colors = [
            'rgba(255, 99, 132, 0.8)',
            'rgba(54, 162, 235, 0.8)',
            'rgba(255, 206, 86, 0.8)',
            'rgba(75, 192, 192, 0.8)',
            'rgba(153, 102, 255, 0.8)'
        ];
        
        window.chartInstances.notificationTypesChart = new Chart(notificationTypesChartElem.getContext('2d'), {
            type: 'pie',
            data: {
                labels: types,
                datasets: [{
                    data: typeData,
                    backgroundColor: colors,
                    borderColor: colors.map(c => c.replace('0.8', '1')),
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: 'Распределение по типам уведомлений'
                    }
                }
            }
        });
    }
    
    if (countryDistributionChartElem) {
        // Пример данных для диаграммы распределения по странам
        const countries = ['Украина', 'Польша', 'США', 'Германия', 'Франция', 'Другие'];
        const countryData = [45, 20, 15, 10, 5, 5];
        const colors = [
            'rgba(255, 206, 86, 0.8)',
            'rgba(75, 192, 192, 0.8)',
            'rgba(153, 102, 255, 0.8)',
            'rgba(255, 159, 64, 0.8)',
            'rgba(54, 162, 235, 0.8)',
            'rgba(201, 203, 207, 0.8)'
        ];
        
        window.chartInstances.countryDistributionChart = new Chart(countryDistributionChartElem.getContext('2d'), {
            type: 'doughnut',
            data: {
                labels: countries,
                datasets: [{
                    data: countryData,
                    backgroundColor: colors,
                    borderColor: colors.map(c => c.replace('0.8', '1')),
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: 'Распределение по странам'
                    }
                }
            }
        });
    }
    
    // Обработчики фильтров периода
    document.querySelectorAll('[data-period]').forEach(button => {
        button.addEventListener('click', function() {
            document.querySelectorAll('[data-period]').forEach(btn => {
                btn.classList.remove('active');
            });
            this.classList.add('active');
            
            // Здесь должен быть код для загрузки статистики за выбранный период
            const period = this.getAttribute('data-period');
            // Заглушка для примера
            if (typeof toastr !== 'undefined') {
                toastr.info(`Загрузка статистики за период: ${period}`);
            }
        });
    });
}

// Функции для работы с API уведомлений

/**
 * Просмотр шаблона уведомления
 * @param {string} id ID шаблона
 */
function viewTemplate(id) {
    // Проверяем, загружается ли уже шаблон
    if (document.getElementById('template-loading-indicator')) {
        console.log('Template loading already in progress');
        return;
    }
    
    // Показываем лоадер
    document.getElementById('viewTemplateLoader').style.display = 'block';
    document.getElementById('viewTemplateContent').style.display = 'none';
    
    // Создаем индикатор загрузки для отслеживания
    const loadingIndicator = document.createElement('div');
    loadingIndicator.id = 'template-loading-indicator';
    loadingIndicator.style.display = 'none';
    document.getElementById('viewTemplateLoader').appendChild(loadingIndicator);
    
    // Открываем модальное окно
    const viewTemplateModal = new bootstrap.Modal(document.getElementById('viewTemplateModal'));
    viewTemplateModal.show();
    
    // Загружаем данные шаблона
    fetch(`/admin/api/notification-templates/${id}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Заполняем поля модального окна
            document.getElementById('view-template-name').textContent = data.name;
            document.getElementById('view-template-type').textContent = data.type === 'manual' ? 'Ручной' : 'Автоматический';
            
            // Остальной код без изменений...
            
            // Скрываем лоадер и показываем контент
            document.getElementById('viewTemplateLoader').style.display = 'none';
            document.getElementById('viewTemplateContent').style.display = 'block';
            
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('template-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
        })
        .catch(error => {
            console.error('Error loading template:', error);
            // Показываем ошибку
            document.getElementById('viewTemplateLoader').innerHTML = 
                `<div class="alert alert-danger">Ошибка загрузки данных: ${error.message}</div>`;
                
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('template-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
        });
}

/**
 * Редактирование шаблона уведомления
 * @param {string} id ID шаблона (если не указан, создается новый)
 */
function editTemplate(id) {
    // Сбрасываем форму
    resetTemplateForm();
    
    // Открываем модальное окно
    const editTemplateModal = new bootstrap.Modal(document.getElementById('editTemplateModal'));
    
    // Если id не указан, значит это создание нового шаблона
    if (!id) {
        document.getElementById('editTemplateModalLabel').textContent = 'Создать шаблон уведомления';
        
        // Для нового шаблона поле типа доступно
        const templateTypeSelect = document.getElementById('template-type');
        if (templateTypeSelect) {
            templateTypeSelect.removeAttribute('disabled');
            templateTypeSelect.style.backgroundColor = '';
        }
        
        editTemplateModal.show();
        return;
    }
    
    // Устанавливаем заголовок для редактирования
    document.getElementById('editTemplateModalLabel').textContent = 'Редактировать шаблон уведомления';
    
    // Добавим индикатор загрузки
    const modalBody = document.querySelector('#editTemplateModal .modal-body');
    if (modalBody) {
        // Удаляем предыдущий индикатор загрузки, если он существует
        const prevLoader = document.getElementById('template-loading-indicator');
        if (prevLoader) {
            prevLoader.remove();
        }
        
        const loadingIndicator = document.createElement('div');
        loadingIndicator.id = 'template-loading-indicator';
        loadingIndicator.className = 'text-center py-3';
        loadingIndicator.innerHTML = '<div class="spinner-border" role="status"><span class="visually-hidden">Загрузка...</span></div>';
        
        // Вставляем индикатор перед формой и скрываем форму
        const form = document.getElementById('templateForm');
        if (form) {
            form.style.display = 'none';
            form.parentNode.insertBefore(loadingIndicator, form);
        }
    }
    
    // Показываем модальное окно перед загрузкой данных
    editTemplateModal.show();
    
    // Проверяем, что запрос на загрузку данных еще не был отправлен
    if (window.isLoadingTemplate) {
        console.log('Загрузка шаблона уже выполняется, пропускаем повторный запрос');
        return;
    }
    
    window.isLoadingTemplate = true;
    
    // Загружаем данные шаблона
    fetch(`/admin/api/notification-templates/${id}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Выводим данные в консоль для отладки
            console.log('Полученные данные шаблона:', data);
            
            // Удаляем индикатор загрузки и показываем форму
            const loadingIndicator = document.getElementById('template-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
            const form = document.getElementById('templateForm');
            if (form) {
                form.style.display = '';
            }
            
            // Заполняем базовые поля формы
            document.getElementById('template-id').value = data.id;
            document.getElementById('template-name').value = data.name;
            
            // Заменяем выпадающий список на нередактируемый текст
            const templateTypeSelect = document.getElementById('template-type');
            const templateTypeValue = data.type;
            
            // Устанавливаем значение, но блокируем взаимодействие
            templateTypeSelect.value = templateTypeValue;
            templateTypeSelect.setAttribute('disabled', 'disabled');
            templateTypeSelect.style.backgroundColor = '#e9ecef';
            
            // Добавляем скрытое поле для отправки значения
            let hiddenInput = document.getElementById('hidden-template-type');
            if (!hiddenInput) {
                hiddenInput = document.createElement('input');
                hiddenInput.type = 'hidden';
                hiddenInput.id = 'hidden-template-type';
                hiddenInput.name = 'type';
                document.getElementById('templateForm').appendChild(hiddenInput);
            }
            hiddenInput.value = templateTypeValue;
            
            // Показываем/скрываем поле события триггера
            const triggerEventContainer = document.getElementById('trigger-event-container');
            triggerEventContainer.style.display = data.type === 'automatic' ? 'block' : 'none';
            
            if (data.type === 'automatic' && data.trigger_event) {
                document.getElementById('trigger-event').value = data.trigger_event;
            }
            
            // Устанавливаем ключи локализации и загружаем значения
            if (data.title_key) {
                console.log('Установка title_key:', data.title_key);
                document.getElementById('title-key').value = data.title_key;
                
                // Загружаем локализации для заголовка
                fetch(`/admin/api/localizations/${data.title_key}`)
                    .then(response => response.json())
                    .then(localizations => {
                        console.log('Полученные локализации заголовка:', localizations);
                        // Заполняем поля локализаций
                        document.getElementById('title-en').value = localizations.en || '';
                        document.getElementById('title-ru').value = localizations.ru || '';
                        document.getElementById('title-uk').value = localizations.uk || '';
                    })
                    .catch(error => {
                        console.error('Error loading title localizations:', error);
                    });
            }
            
            if (data.message_key) {
                console.log('Установка message_key:', data.message_key);
                document.getElementById('message-key').value = data.message_key;
                
                // Загружаем локализации для сообщения
                fetch(`/admin/api/localizations/${data.message_key}`)
                    .then(response => response.json())
                    .then(localizations => {
                        console.log('Полученные локализации сообщения:', localizations);
                        // Заполняем поля локализаций
                        document.getElementById('message-en').value = localizations.en || '';
                        document.getElementById('message-ru').value = localizations.ru || '';
                        document.getElementById('message-uk').value = localizations.uk || '';
                    })
                    .catch(error => {
                        console.error('Error loading message localizations:', error);
                    });
            }
            
            // Заполняем URL изображений для разных языков
            if (data.image) {
                document.getElementById('edit-image-en-input').value = data.image.en || '';
                document.getElementById('edit-image-ru-input').value = data.image.ru || '';
                document.getElementById('edit-image-uk-input').value = data.image.uk || '';
            }
            
            // Проверяем наличие данных кнопки
            const hasButton = !!(data.button_text_key || (data.button && data.button.text_key));
            const buttonTextKey = data.button_text_key || (data.button && data.button.text_key) || '';
            const buttonUrl = (data.button && data.button.url) || '';
            const buttonCallback = (data.button && data.button.callback) || '';
            
            document.getElementById('has-button').checked = hasButton;
            document.getElementById('button-settings').style.display = hasButton ? 'block' : 'none';
            
            if (hasButton) {
                console.log('Установка button_text_key:', buttonTextKey);
                document.getElementById('button-text-key').value = buttonTextKey;
                document.getElementById('edit-button-url').value = buttonUrl;
                document.getElementById('edit-button-callback').value = buttonCallback;
                
                // Загружаем локализации для кнопки
                if (buttonTextKey) {
                    fetch(`/admin/api/localizations/${buttonTextKey}`)
                        .then(response => response.json())
                        .then(localizations => {
                            console.log('Полученные локализации кнопки:', localizations);
                            // Заполняем поля локализаций
                            document.getElementById('button-text-en').value = localizations.en || '';
                            document.getElementById('button-text-ru').value = localizations.ru || '';
                            document.getElementById('button-text-uk').value = localizations.uk || '';
                        })
                        .catch(error => {
                            console.error('Error loading button localizations:', error);
                        });
                }
            }

            // Устанавливаем статус активности
            document.getElementById('template-active').checked = data.active !== false; // По умолчанию true, если не указано явно
            
            // Выводим в консоль состояние полей после заполнения
            setTimeout(() => {
                console.log('Состояние полей формы после заполнения:', {
                    'title-key': document.getElementById('title-key').value,
                    'message-key': document.getElementById('message-key').value,
                    'button-text-key': document.getElementById('button-text-key').value,
                    'title-en': document.getElementById('title-en').value,
                    'message-en': document.getElementById('message-en').value
                });
            }, 500);
            
            // Сбрасываем флаг загрузки
            window.isLoadingTemplate = false;
        })
        .catch(error => {
            console.error('Error loading template for edit:', error);
            
            // Показываем сообщение об ошибке
            showNotification('Ошибка загрузки данных шаблона: ' + error.message, 'danger');
            
            // Сбрасываем флаг загрузки
            window.isLoadingTemplate = false;
            
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('template-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
            
            // Показываем форму, чтобы пользователь мог попробовать ещё раз
            const form = document.getElementById('templateForm');
            if (form) {
                form.style.display = '';
            }
        });
}

/**
 * Загрузка локализаций для предпросмотра
 * @param {string} previewContainerId ID контейнера для предпросмотра
 * @param {string} key Ключ локализации
 * @returns {Promise} Promise, который разрешается, когда запрос завершен
 */
function loadLocalizations(previewContainerId, key) {
    const previewContainer = document.getElementById(previewContainerId);
    
    if (!previewContainer) return Promise.resolve();
    
    previewContainer.innerHTML = '<div class="text-center"><div class="spinner-border spinner-border-sm" role="status"></div> Загрузка...</div>';
    
    return fetch(`/admin/api/localizations/${key}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            let html = '';
            
            if (!data || Object.keys(data).length === 0) {
                html = '<div class="alert alert-warning">Локализации не найдены</div>';
            } else {
                html = '<div class="row">';
                
                // Добавляем предпросмотр для каждого языка
                if (data.en) {
                    html += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">English</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.en}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                if (data.ru) {
                    html += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Русский</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.ru}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                if (data.uk) {
                    html += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Українська</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.uk}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                html += '</div>';
            }
            
            previewContainer.innerHTML = html;
        })
        .catch(error => {
            console.error('Error loading localizations:', error);
            previewContainer.innerHTML = '<div class="alert alert-danger">Ошибка загрузки локализаций</div>';
        });
}

/**
 * Инициализация обработчиков для ключей локализации
 */
function initLocalizationKeyHandlers() {
    // Сначала удалим все существующие обработчики
    document.getElementById('title-key').removeEventListener('change', loadTitleLocalizations);
    document.getElementById('message-key').removeEventListener('change', loadMessageLocalizations);
    document.getElementById('button-text-key').removeEventListener('change', loadButtonLocalizations);

    // Добавим флаги для отслеживания состояния загрузки
    window.loadingLocalizationsTitle = false;
    window.loadingLocalizationsMessage = false;
    window.loadingLocalizationsButton = false;

    // Функции для загрузки локализаций по разным ключам
    function loadTitleLocalizations() {
        const key = this.value;
        if (!key || window.loadingLocalizationsTitle) return;
        
        window.loadingLocalizationsTitle = true;
        loadLocalizations('title-preview', key).finally(() => {
            window.loadingLocalizationsTitle = false;
        });
    }

    function loadMessageLocalizations() {
        const key = this.value;
        if (!key || window.loadingLocalizationsMessage) return;
        
        window.loadingLocalizationsMessage = true;
        loadLocalizations('message-preview', key).finally(() => {
            window.loadingLocalizationsMessage = false;
        });
    }

    function loadButtonLocalizations() {
        const key = this.value;
        if (!key || window.loadingLocalizationsButton) return;
        
        window.loadingLocalizationsButton = true;
        loadLocalizations('button-preview', key).finally(() => {
            window.loadingLocalizationsButton = false;
        });
    }

    // Обработчик изменения ключа заголовка
    document.getElementById('title-key').addEventListener('change', loadTitleLocalizations);
    
    // Обработчик изменения ключа сообщения
    document.getElementById('message-key').addEventListener('change', loadMessageLocalizations);
    
    // Обработчик изменения ключа кнопки
    document.getElementById('button-text-key').addEventListener('change', loadButtonLocalizations);
}

// Добавим вызов инициализации при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Инициализация уведомлений, если мы находимся на странице уведомлений
    if (document.querySelector('.notifications-page')) {
        initNotifications();
        
        // Инициализация новых обработчиков
        initLocalizationKeyHandlers();
        initLocalizationCheckButtons();
    }
});

/**
 * Сброс формы редактирования шаблона
 */
function resetTemplateForm() {
    // Очищаем все поля формы
    document.getElementById('template-id').value = '';
    document.getElementById('template-name').value = '';
    document.getElementById('template-type').value = 'manual';
    
    // Удаляем скрытое поле типа, если оно есть
    const hiddenType = document.getElementById('hidden-template-type');
    if (hiddenType) {
        hiddenType.remove();
    }
    
    // Разблокируем поле типа для нового шаблона
    const templateTypeSelect = document.getElementById('template-type');
    if (templateTypeSelect) {
        templateTypeSelect.removeAttribute('disabled');
        templateTypeSelect.style.backgroundColor = '';
    }
    
    // Скрываем поле события триггера
    document.getElementById('trigger-event-container').style.display = 'none';
    
    // Очищаем ключи локализации
    document.getElementById('title-key').value = '';
    document.getElementById('message-key').value = '';
    
    // Очищаем URL изображений для разных языков
    document.getElementById('edit-image-en-input').value = '';
    document.getElementById('edit-image-ru-input').value = '';
    document.getElementById('edit-image-uk-input').value = '';
    
    // Выключаем переключатель кнопки
    document.getElementById('has-button').checked = false;
    
    // Скрываем настройки кнопки
    document.getElementById('button-settings').style.display = 'none';
    
    // Очищаем ключ локализации кнопки
    document.getElementById('button-text-key').value = '';
    
    // Очищаем URL и Callback кнопки
    document.getElementById('edit-button-url').value = '';
    document.getElementById('edit-button-callback').value = '';
    
    // Активный шаблон по умолчанию
    document.getElementById('template-active').checked = true;
}

document.addEventListener('DOMContentLoaded', function() {
  // Обработчик для удаления backdrop после закрытия модального окна
  document.querySelectorAll('.modal').forEach(function(modal) {
    modal.addEventListener('hidden.bs.modal', function() {
      // Удаляем все .modal-backdrop элементы
      const backdrops = document.querySelectorAll('.modal-backdrop');
      backdrops.forEach(backdrop => {
        backdrop.remove();
      });
      
      // Убираем класс modal-open с body
      document.body.classList.remove('modal-open');
      document.body.style.overflow = '';
      document.body.style.paddingRight = '';
    });
  });
});

/**
 * Загрузка списка стран для фильтра
 */
function loadCountriesForFilter() {
    const countriesList = document.getElementById('countries-list');
    
    // Показываем индикатор загрузки
    countriesList.innerHTML = '<div class="text-center py-3"><div class="spinner-border" role="status"><span class="visually-hidden">Загрузка...</span></div></div>';
    
    // Загружаем список стран с сервера
    fetch('/admin/api/countries-with-users')
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Очищаем список
            countriesList.innerHTML = '';
            
            // Проверяем, есть ли данные
            if (!data || data.length === 0) {
                countriesList.innerHTML = '<div class="alert alert-info">Нет доступных стран</div>';
                return;
            }
            
            // Добавляем опцию "Все страны"
            const allCountriesDiv = document.createElement('div');
            allCountriesDiv.className = 'mb-2';
            allCountriesDiv.innerHTML = `
                <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="country-all" value="all" checked>
                    <label class="form-check-label" for="country-all">
                        🌎 Все страны
                    </label>
                </div>
            `;
            countriesList.appendChild(allCountriesDiv);
            
            // Сортируем страны по количеству пользователей
            data.sort((a, b) => b.count - a.count);
            
            // Добавляем все страны в список
            data.forEach(country => {
                const countryDiv = document.createElement('div');
                countryDiv.className = 'mb-2';
                countryDiv.innerHTML = `
                    <div class="form-check">
                        <input class="form-check-input country-checkbox" type="checkbox" id="country-${country.code}" value="${country.code}" data-users="${country.count}" disabled>
                        <label class="form-check-label" for="country-${country.code}">
                            ${country.emoji || ''} ${country.name || country.code} (${country.count} пользователей)
                        </label>
                    </div>
                `;
                countriesList.appendChild(countryDiv);
            });
            
            // Добавляем обработчик для чекбокса "Все страны"
            const allCountriesCheckbox = document.getElementById('country-all');
            if (allCountriesCheckbox) {
                allCountriesCheckbox.addEventListener('change', function() {
                    const checked = this.checked;
                    document.querySelectorAll('.country-checkbox').forEach(checkbox => {
                        checkbox.disabled = checked;
                        if (!checked) {
                            checkbox.checked = false;
                        }
                    });
                });
            }
            
            // Добавляем поиск по странам
            const searchCountriesInput = document.getElementById('search-countries');
            if (searchCountriesInput) {
                searchCountriesInput.disabled = false;
                searchCountriesInput.addEventListener('keyup', function() {
                    const value = this.value.toLowerCase();
                    document.querySelectorAll('#countries-list .form-check').forEach(item => {
                        const text = item.textContent.toLowerCase();
                        if (item.querySelector('#country-all')) {
                            // Всегда показываем опцию "Все страны"
                            item.style.display = '';
                        } else {
                            item.style.display = text.includes(value) ? '' : 'none';
                        }
                    });
                });
            }
        })
        .catch(error => {
            console.error('Error loading countries:', error);
            countriesList.innerHTML = `
                <div class="alert alert-danger">
                    Ошибка загрузки списка стран: ${error.message}
                    <button type="button" class="btn-close float-end" data-bs-dismiss="alert" aria-label="Close"></button>
                </div>
            `;
        });
}
