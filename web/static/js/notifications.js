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
        showNotification('Шаблон успешно удален', 'success');
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    })
    .catch(error => {
        console.error('Error deleting template:', error);
        showNotification(error.error || 'Ошибка удаления шаблона', 'danger');
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
 * Просмотр задачи на отправку уведомления
 * @param {string} id ID задачи
 */
function viewTask(id) {
    // Проверяем, выполняется ли уже загрузка
    if (document.getElementById('view-task-loading-indicator')) {
        console.log('Task view loading already in progress, ignoring duplicate request');
        return;
    }
    
    // Показываем лоадер
    document.getElementById('viewTaskLoader').style.display = 'block';
    document.getElementById('viewTaskContent').style.display = 'none';
    document.getElementById('cancel-task-btn').style.display = 'none';
    
    // Добавляем индикатор загрузки для отслеживания
    const loadingIndicator = document.createElement('div');
    loadingIndicator.id = 'view-task-loading-indicator';
    loadingIndicator.style.display = 'none';
    document.getElementById('viewTaskLoader').appendChild(loadingIndicator);
    
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
            // Определяем функцию форматирования даты
            const formatDate = (dateStr) => {
                if (!dateStr) return '-';
                try {
                    const date = new Date(dateStr);
                    // Проверяем валидность даты
                    if (isNaN(date.getTime())) return '-';
                    return date.toLocaleString();
                } catch (e) {
                    return '-';
                }
            };
            
            // Заполняем общую информацию
            const templateName = data.template ? data.template.name : 'Неизвестный шаблон';
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
            const targetType = data.target_type || data.targetType || "";
            let targetTypeText = '';
            switch (targetType) {
                case 'all':
                    targetTypeText = 'Все пользователи';
                    break;
                case 'country':
                    targetTypeText = 'По странам';
                    break;
                case 'activity':
                    targetTypeText = 'По активности';
                    break;
                case 'custom':
                    targetTypeText = 'Выборочно';
                    break;
                default:
                    targetTypeText = targetType || 'Неизвестно';
                    break;
            }
            document.getElementById('view-task-target-type').textContent = targetTypeText;
            
            // Отображаем запланированное время
            const scheduledTime = data.scheduled_at || data.scheduledAt;
            if (scheduledTime) {
                document.getElementById('view-task-scheduled-at').textContent = formatDate(scheduledTime);
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
            // Специальная обработка для поддержки разных форматов API
            let totalUsers = 0;
            let sentCount = 0;
            let deliveredCount = 0;
            let readCount = 0;
            
            // Ищем поля по разным именам
            for (const key in data) {
                const keyLower = key.toLowerCase();
                if (keyLower.includes('total') && keyLower.includes('user')) {
                    totalUsers = parseInt(data[key]) || 0;
                }
                if (keyLower.includes('sent') && keyLower.includes('count')) {
                    sentCount = parseInt(data[key]) || 0;
                }
                if (keyLower.includes('deliver') && keyLower.includes('count')) {
                    deliveredCount = parseInt(data[key]) || 0;
                }
                if (keyLower.includes('read') && keyLower.includes('count')) {
                    readCount = parseInt(data[key]) || 0;
                }
            }
            
            // Прямое обращение к предполагаемым именам полей
            if (totalUsers === 0) totalUsers = parseInt(data.totalUsers || data.total_users || 0);
            if (sentCount === 0) sentCount = parseInt(data.sentCount || data.sent_count || 0);
            if (deliveredCount === 0) deliveredCount = parseInt(data.deliveredCount || data.delivered_count || 0);
            if (readCount === 0) readCount = parseInt(data.readCount || data.read_count || 0);
            
            document.getElementById('view-task-total-users').textContent = totalUsers;
            document.getElementById('view-task-sent-count').textContent = sentCount;
            document.getElementById('view-task-delivered-count').textContent = deliveredCount;
            document.getElementById('view-task-read-count').textContent = readCount;
            
            // Отображаем оставшееся время (если задача в процессе)
            const timeRemainingContainer = document.getElementById('view-task-time-remaining-container');
            if (data.status === 'processing' && data.estimatedTimeRemaining) {
                timeRemainingContainer.style.display = 'block';
                document.getElementById('view-task-time-remaining').textContent = data.estimatedTimeRemaining;
            } else {
                timeRemainingContainer.style.display = 'none';
            }
            
            // Отображаем параметры таргетинга
            const targetParams = data.target_params || data.targetParams || {};
            let targetParamsHTML = '';
            
            switch (targetType) {
                case 'all':
                    targetParamsHTML = '<p>Все пользователи</p>';
                    break;
                    
                case 'country':
                    const countries = targetParams.countries || [];
                    if (countries && countries.length > 0) {
                        targetParamsHTML = '<p><strong>Выбранные страны:</strong></p><ul>';
                        countries.forEach(country => {
                            targetParamsHTML += `<li>${country}</li>`;
                        });
                        targetParamsHTML += '</ul>';
                    } else {
                        targetParamsHTML = '<p>Все страны</p>';
                    }
                    break;
                    
                case 'activity':
                    const activityFilters = targetParams.activity_filters || targetParams.activityFilters || [];
                    if (activityFilters && activityFilters.length > 0) {
                        const filterLabels = {
                            'inactive_3days': 'Не играл менее 3 дней (от 3 дней до 12 часов)',
                            'inactive_7days': 'Не играл более 3 дней и менее 7 дней',
                            'inactive_14days': 'Не играл более 7 дней и менее 14 дней',
                            'inactive_more_14days': 'Не играл более 14 дней'
                        };
                        
                        targetParamsHTML = '<p><strong>Фильтры активности:</strong></p><ul>';
                        activityFilters.forEach(filter => {
                            const label = filterLabels[filter] || filter;
                            targetParamsHTML += `<li>${label}</li>`;
                        });
                        targetParamsHTML += '</ul>';
                    }
                    break;
                    
                case 'custom':
                    targetParamsHTML = `<p><strong>Выбранные пользователи:</strong> ${totalUsers} чел.</p>`;
                    break;
                    
                default:
                    targetParamsHTML = `<p><strong>Тип таргетинга:</strong> ${targetType}</p>`;
            }
            
            document.getElementById('view-task-target-params').innerHTML = targetParamsHTML;
            
            // Загрузка получателей и их макросов
            if (id) {
                // Показываем секцию макросов
                document.getElementById('user-macros-section').style.display = 'block';
                
                // Получаем список получателей для этой задачи
                fetch(`/admin/api/notification-recipients?task_id=${id}&limit=100`)
                    .then(response => {
                        if (!response.ok) {
                            throw new Error(`HTTP error ${response.status}`);
                        }
                        return response.json();
                    })
                    .then(recipientsData => {
                        const recipients = recipientsData.recipients || [];
                        if (recipients && recipients.length > 0) {
                            // Создаем таблицу для отображения макросов
                            let macrosHTML = '<table class="table table-sm table-bordered">';
                            macrosHTML += '<thead><tr><th>ID пользователя</th><th>Макросы</th><th>Статус</th></tr></thead>';
                            macrosHTML += '<tbody>';
                            
                            // Проверяем, есть ли макросы хотя бы у одного получателя
                            let hasMacros = false;
                            
                            recipients.forEach(recipient => {
                                let macrosText = '';
                                let macrosObj = null;
                                
                                // Пытаемся получить макросы из разных возможных мест
                                if (recipient.macros) {
                                    try {
                                        // Если macros - строка, пытаемся распарсить JSON
                                        if (typeof recipient.macros === 'string') {
                                            macrosObj = JSON.parse(recipient.macros);
                                        } 
                                        // Если macros уже объект
                                        else if (typeof recipient.macros === 'object') {
                                            macrosObj = recipient.macros;
                                        }
                                        
                                        // Форматируем макросы в читаемый вид
                                        if (macrosObj && Object.keys(macrosObj).length > 0) {
                                            hasMacros = true;
                                            macrosText = '';
                                            for (const key in macrosObj) {
                                                macrosText += `<span class="badge bg-info text-dark me-1 mb-1">${key}: ${macrosObj[key]}</span>`;
                                            }
                                        } else {
                                            macrosText = '<span class="text-muted">—</span>';
                                        }
                                    } catch (e) {
                                        macrosText = '<span class="text-muted">—</span>';
                                    }
                                } else {
                                    macrosText = '<span class="text-muted">—</span>';
                                }
                                
                                // Определяем класс и текст статуса
                                let statusClass = 'secondary';
                                let statusText = recipient.status || 'pending';
                                
                                switch (statusText) {
                                    case 'sent':
                                        statusClass = 'primary';
                                        break;
                                    case 'delivered':
                                        statusClass = 'info';
                                        break;
                                    case 'read':
                                        statusClass = 'success';
                                        break;
                                    case 'failed':
                                        statusClass = 'danger';
                                        break;
                                }
                                
                                macrosHTML += `<tr>
                                    <td>${recipient.user_id}</td>
                                    <td>${macrosText}</td>
                                    <td><span class="badge bg-${statusClass}">${statusText}</span></td>
                                </tr>`;
                            });
                            
                            macrosHTML += '</tbody></table>';
                            
                            // Если получателей много, добавляем ограничение по высоте и скролл
                            if (recipients.length > 5) {
                                macrosHTML = `<div style="max-height: 300px; overflow-y: auto;">${macrosHTML}</div>`;
                            }
                            
                            // Добавляем заголовок с общей информацией
                            const heading = `<h6 class="mb-3">Получатели (${recipients.length} из ${recipientsData.total || recipients.length})</h6>`;
                            
                            document.getElementById('view-task-user-macros').innerHTML = heading + macrosHTML;
                        } else {
                            document.getElementById('view-task-user-macros').innerHTML = '<div class="alert alert-info">Нет данных о получателях</div>';
                        }
                    })
                    .catch(error => {
                        document.getElementById('view-task-user-macros').innerHTML = 
                            `<div class="alert alert-danger">Ошибка загрузки данных о получателях: ${error.message}</div>`;
                    });
            } else {
                // Скрываем секцию макросов, если id задачи не указан
                document.getElementById('user-macros-section').style.display = 'none';
            }
            
            // Отображаем время выполнения
            document.getElementById('view-task-created-at').textContent = formatDate(data.created_at);
            document.getElementById('view-task-started-at').textContent = formatDate(data.started_at);
            document.getElementById('view-task-completed-at').textContent = formatDate(data.completed_at);
            
            // Скрываем лоадер и показываем контент
            document.getElementById('viewTaskLoader').style.display = 'none';
            document.getElementById('viewTaskContent').style.display = 'block';
            
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('view-task-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
        })
        .catch(error => {
            // Показываем ошибку в лоадере
            document.getElementById('viewTaskLoader').innerHTML = `
            <div class="alert alert-danger">
                Ошибка загрузки данных: ${error.message}
                <button type="button" class="btn-close float-end" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
            `;
            
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('view-task-loading-indicator');
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
        });
}

/**
 * Отмена задачи на отправку уведомления
 * @param {string} id ID задачи
 */
function cancelTask(id) {
    // Проверяем, выполняется ли уже отмена
    if (document.getElementById('cancel-task-loading-indicator')) {
        console.log('Task cancellation already in progress, ignoring duplicate request');
        return;
    }
    
    // Добавляем индикатор загрузки
    const cancelBtn = document.querySelector(`[data-id="${id}"].cancel-task`) || document.getElementById('cancel-task-btn');
    if (cancelBtn) {
        cancelBtn.disabled = true;
        cancelBtn.innerHTML = '<span id="cancel-task-loading-indicator" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Отмена...';
    }
    
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
        showNotification('Задача успешно отменена', 'success');
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    })
    .catch(error => {
        console.error('Error canceling task:', error);
        showNotification(error.error || 'Ошибка отмены задачи', 'danger');
    })
    .finally(() => {
        // Восстанавливаем кнопку
        const cancelBtn = document.querySelector(`[data-id="${id}"].cancel-task`) || document.getElementById('cancel-task-btn');
        if (cancelBtn) {
            cancelBtn.disabled = false;
            cancelBtn.innerHTML = 'Отменить задачу';
        }
        
        // Удаляем индикатор загрузки
        const indicator = document.getElementById('cancel-task-loading-indicator');
        if (indicator) {
            indicator.remove();
        }
    });
}

/**
 * Создание задачи уведомления
 */
function createNotificationTask() {
    // Проверяем, выполняется ли уже создание задачи
    if (document.getElementById('task-loading-indicator')) {
        console.log('Task creation already in progress, ignoring duplicate request');
        return;
    }
    
    // Добавляем индикатор загрузки для отслеживания
    const createTaskBtn = document.getElementById('create-task-btn');
    if (createTaskBtn) {
        createTaskBtn.disabled = true;
        createTaskBtn.innerHTML = '<span id="task-loading-indicator" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Создание...';
    }
    
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
                showNotification('Пожалуйста, выберите хотя бы одну страну', 'danger');
                resetCreateButton();
                return;
            }
            
            targetParams = { countries: selectedCountries };
        }
    } else if (targetType === 'activity') {
        // Получаем выбранный фильтр активности из select
        const activityFilter = document.getElementById('activity-filter-select').value;
        
        if (!activityFilter) {
            showNotification('Пожалуйста, выберите фильтр активности', 'danger');
            resetCreateButton();
            return;
        }
        
        targetParams = { activityFilters: [activityFilter] };
    }
    
    // Проверяем, включена ли отложенная отправка
    const scheduledSend = document.getElementById('scheduled-send').checked;
    let scheduledAt = null;
    
    if (scheduledSend) {
        scheduledAt = document.getElementById('scheduled-at').value;
        
        if (!scheduledAt) {
            showNotification('Пожалуйста, укажите дату и время отправки', 'danger');
            resetCreateButton();
            return;
        }
    } else {
        // Для немедленной отправки используем текущее время
        const now = new Date();
        scheduledAt = now.toISOString();
    }
    
    // Проверяем, включена ли адаптация времени
    const adjustTime = document.getElementById('adjust-time');
    if (adjustTime && adjustTime.checked) {
        const sendTimeStart = document.getElementById('send-time-start').value;
        const sendTimeEnd = document.getElementById('send-time-end').value;
        
        if (sendTimeStart && sendTimeEnd) {
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
        scheduledAt: scheduledAt,
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
        showNotification(`Задача на отправку уведомления создана успешно. Всего получателей: ${result.totalUsers}`, 'success');
        
        // Обновляем страницу
        setTimeout(() => {
            window.location.reload();
        }, 1500);
    })
    .catch(error => {
        console.error('Error creating task:', error);
        showNotification(error.error || 'Ошибка создания задачи', 'danger');
    })
    .finally(() => {
        resetCreateButton();
    });
    
    function resetCreateButton() {
        const createTaskBtn = document.getElementById('create-task-btn');
        if (createTaskBtn) {
            createTaskBtn.disabled = false;
            createTaskBtn.innerHTML = 'Создать задачу';
        }
        
        const indicator = document.getElementById('task-loading-indicator');
        if (indicator) {
            indicator.remove();
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

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Инициализация обработчиков для шаблонов уведомлений
    initializeTemplateHandlers();
    
    // Инициализация форм
    initializeNotificationForms();

    // Для модального окна просмотра задачи
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
 * Инициализация обработчиков для шаблонов уведомлений
 */
function initializeTemplateHandlers() {
    // Добавляем новые обработчики для кнопок
    document.querySelectorAll('.view-template').forEach(button => {
        button.addEventListener('click', function() {
            const templateId = this.getAttribute('data-id');
            viewTemplate(templateId);
        });
    });
    
    document.querySelectorAll('.edit-template').forEach(button => {
        button.addEventListener('click', function() {
            const templateId = this.getAttribute('data-id');
            editTemplate(templateId);
        });
    });
    
    document.querySelectorAll('.create-task').forEach(button => {
        button.addEventListener('click', function() {
            const templateId = this.getAttribute('data-id');
            const templateName = this.getAttribute('data-name');
            createTask(templateId, templateName);
        });
    });
    
    document.querySelectorAll('.delete-template').forEach(button => {
        button.addEventListener('click', function() {
            const templateId = this.getAttribute('data-id');
            if (confirm('Вы уверены, что хотите удалить этот шаблон?')) {
                deleteTemplate(templateId);
            }
        });
    });
    
    document.querySelectorAll('.view-task').forEach(button => {
        button.addEventListener('click', function() {
            const taskId = this.getAttribute('data-id');
            viewTask(taskId);
        });
    });
    
    document.querySelectorAll('.cancel-task').forEach(button => {
        button.addEventListener('click', function() {
            const taskId = this.getAttribute('data-id');
            if (confirm('Вы уверены, что хотите отменить эту задачу?')) {
                cancelTask(taskId);
            }
        });
    });
    
    // Кнопка создания нового шаблона
    const createTemplateBtn = document.getElementById('create-template-btn');
    if (createTemplateBtn) {
        createTemplateBtn.addEventListener('click', function() {
            resetTemplateForm();
        });
    }
    
    // Обработчик отмены задачи в модальном окне просмотра
    const cancelTaskBtn = document.getElementById('cancel-task-btn');
    if (cancelTaskBtn) {
        cancelTaskBtn.addEventListener('click', function() {
            const taskId = this.getAttribute('data-id');
            if (confirm('Вы уверены, что хотите отменить эту задачу?')) {
                cancelTask(taskId);
            }
        });
    }
}

/**
 * Инициализация форм для уведомлений
 */
function initializeNotificationForms() {
    // Инициализация формы шаблона
    const templateForm = document.getElementById('templateForm');
    if (templateForm) {
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
    }
    
    // Переключатель для настроек кнопки
    const hasButtonCheckbox = document.getElementById('has-button');
    if (hasButtonCheckbox) {
        hasButtonCheckbox.addEventListener('change', function() {
            const buttonSettings = document.getElementById('button-settings');
            if (buttonSettings) {
                buttonSettings.style.display = this.checked ? 'block' : 'none';
            }
        });
    }
}// web/static/js/notifications.js - упрощенная версия

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
            console.log('Полученные данные шаблона:', data);
            
            // Заполняем основную информацию
            document.getElementById('view-template-name').textContent = data.name || 'Без названия';
            document.getElementById('view-template-type').textContent = data.type === 'manual' ? 'Ручной' : 'Автоматический';
            
            // Заполняем информацию о заголовке
            const titleKeyElem = document.getElementById('view-title-key');
            const titleLocalizationsElem = document.getElementById('view-title-localizations');
            
            if (titleKeyElem) titleKeyElem.textContent = data.title_key || '';
            
            if (titleLocalizationsElem && data.title) {
                let titleHTML = '<div class="row">';
                if (data.title.en) {
                    titleHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">English</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.title.en}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                if (data.title.ru) {
                    titleHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Русский</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.title.ru}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                if (data.title.uk) {
                    titleHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Українська</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.title.uk}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                titleHTML += '</div>';
                titleLocalizationsElem.innerHTML = titleHTML;
            }
            
            // Заполняем информацию о сообщении
            const messageKeyElem = document.getElementById('view-message-key');
            const messageLocalizationsElem = document.getElementById('view-message-localizations');
            
            if (messageKeyElem) messageKeyElem.textContent = data.message_key || '';
            
            if (messageLocalizationsElem && data.message) {
                let messageHTML = '<div class="row">';
                if (data.message.en) {
                    messageHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">English</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.message.en}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                if (data.message.ru) {
                    messageHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Русский</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.message.ru}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                if (data.message.uk) {
                    messageHTML += `
                        <div class="col-md-4">
                            <div class="card">
                                <div class="card-header">Українська</div>
                                <div class="card-body">
                                    <p class="mb-0">${data.message.uk}</p>
                                </div>
                            </div>
                        </div>
                    `;
                }
                messageHTML += '</div>';
                messageLocalizationsElem.innerHTML = messageHTML;
            }
            
            // Заполняем информацию о кнопке, если она есть
            const buttonSection = document.getElementById('button-section');
            if (data.button && (data.button.text || data.button.url || data.button.callback)) {
                buttonSection.style.display = 'block';
                
                const buttonKeyElem = document.getElementById('view-button-key');
                const buttonLocalizationsElem = document.getElementById('view-button-localizations');
                const buttonUrlElem = document.getElementById('view-button-url');
                const buttonCallbackElem = document.getElementById('view-button-callback');
                
                if (buttonKeyElem) buttonKeyElem.textContent = data.button.text_key || '';
                
                if (buttonUrlElem) buttonUrlElem.textContent = data.button.url || '';
                if (buttonCallbackElem) buttonCallbackElem.textContent = data.button.callback || '';
                
                if (buttonLocalizationsElem && data.button.text) {
                    let buttonHTML = '<div class="row">';
                    if (data.button.text.en) {
                        buttonHTML += `
                            <div class="col-md-4">
                                <div class="card">
                                    <div class="card-header">English</div>
                                    <div class="card-body">
                                        <p class="mb-0">${data.button.text.en}</p>
                                    </div>
                                </div>
                            </div>
                        `;
                    }
                    if (data.button.text.ru) {
                        buttonHTML += `
                            <div class="col-md-4">
                                <div class="card">
                                    <div class="card-header">Русский</div>
                                    <div class="card-body">
                                        <p class="mb-0">${data.button.text.ru}</p>
                                    </div>
                                </div>
                            </div>
                        `;
                    }
                    if (data.button.text.uk) {
                        buttonHTML += `
                            <div class="col-md-4">
                                <div class="card">
                                    <div class="card-header">Українська</div>
                                    <div class="card-body">
                                        <p class="mb-0">${data.button.text.uk}</p>
                                    </div>
                                </div>
                            </div>
                        `;
                    }
                    buttonHTML += '</div>';
                    buttonLocalizationsElem.innerHTML = buttonHTML;
                }
            } else {
                buttonSection.style.display = 'none';
            }
            
            // Заполняем информацию об изображении, если оно есть
            const imageSection = document.getElementById('image-section');
            if (data.image && Object.keys(data.image).length > 0) {
                imageSection.style.display = 'block';
                
                let imageHTML = '<div class="row">';
                
                if (data.image.en) {
                    imageHTML += `
                        <div class="col-md-4">
                            <div class="card mb-3">
                                <div class="card-header">English</div>
                                <div class="card-body text-center">
                                    <img src="${data.image.en}" class="img-fluid" style="max-height: 200px;" alt="English Image">
                                </div>
                                <div class="card-footer">
                                    <a href="${data.image.en}" target="_blank" class="btn btn-sm btn-outline-primary">
                                        <i class="bi bi-box-arrow-up-right me-1"></i> Открыть
                                    </a>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                if (data.image.ru) {
                    imageHTML += `
                        <div class="col-md-4">
                            <div class="card mb-3">
                                <div class="card-header">Русский</div>
                                <div class="card-body text-center">
                                    <img src="${data.image.ru}" class="img-fluid" style="max-height: 200px;" alt="Russian Image">
                                </div>
                                <div class="card-footer">
                                    <a href="${data.image.ru}" target="_blank" class="btn btn-sm btn-outline-primary">
                                        <i class="bi bi-box-arrow-up-right me-1"></i> Открыть
                                    </a>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                if (data.image.uk) {
                    imageHTML += `
                        <div class="col-md-4">
                            <div class="card mb-3">
                                <div class="card-header">Українська</div>
                                <div class="card-body text-center">
                                    <img src="${data.image.uk}" class="img-fluid" style="max-height: 200px;" alt="Ukrainian Image">
                                </div>
                                <div class="card-footer">
                                    <a href="${data.image.uk}" target="_blank" class="btn btn-sm btn-outline-primary">
                                        <i class="bi bi-box-arrow-up-right me-1"></i> Открыть
                                    </a>
                                </div>
                            </div>
                        </div>
                    `;
                }
                
                imageHTML += '</div>';
                imageSection.querySelector('.card-body').innerHTML = imageHTML;
            } else {
                imageSection.style.display = 'none';
            }
            
            // Настраиваем кнопку редактирования
            const editBtn = document.getElementById('edit-template-btn');
            if (editBtn) {
                editBtn.setAttribute('data-id', data.id);
                editBtn.onclick = function() {
                    // Закрываем текущее модальное окно
                    viewTemplateModal.hide();
                    
                    // Открываем окно редактирования
                    setTimeout(() => {
                        editTemplate(data.id);
                    }, 500);
                };
            }
            
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
    
    // Проверяем, что запрос на загрузку данных еще не был отправлен
    if (document.getElementById('edit-template-loading-indicator')) {
        console.log('Edit template loading already in progress');
        return;
    }
    
    // Добавим индикатор загрузки
    const modalBody = document.querySelector('#editTemplateModal .modal-body');
    if (modalBody) {
        // Удаляем предыдущий индикатор загрузки, если он существует
        const prevLoader = document.getElementById('edit-template-loading-indicator');
        if (prevLoader) {
            prevLoader.remove();
        }
        
        const loadingIndicator = document.createElement('div');
        loadingIndicator.id = 'edit-template-loading-indicator';
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
    
    // Загружаем данные шаблона
    fetch(`/admin/api/notification-templates/${id}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            // Удаляем индикатор загрузки и показываем форму
            const loadingIndicator = document.getElementById('edit-template-loading-indicator');
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
                document.getElementById('title-key').value = data.title_key;
                
                // Загружаем локализации для заголовка
                fetch(`/admin/api/localizations/${data.title_key}`)
                    .then(response => response.json())
                    .then(localizations => {
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
                document.getElementById('message-key').value = data.message_key;
                
                // Загружаем локализации для сообщения
                fetch(`/admin/api/localizations/${data.message_key}`)
                    .then(response => response.json())
                    .then(localizations => {
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
                document.getElementById('button-text-key').value = buttonTextKey;
                document.getElementById('edit-button-url').value = buttonUrl;
                document.getElementById('edit-button-callback').value = buttonCallback;
                
                // Загружаем локализации для кнопки
                if (buttonTextKey) {
                    fetch(`/admin/api/localizations/${buttonTextKey}`)
                        .then(response => response.json())
                        .then(localizations => {
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
        })
        .catch(error => {
            console.error('Error loading template for edit:', error);
            
            // Показываем сообщение об ошибке
            showNotification('Ошибка загрузки данных шаблона: ' + error.message, 'danger');
            
            // Удаляем индикатор загрузки
            const loadingIndicator = document.getElementById('edit-template-loading-indicator');
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
    
    // Очищаем поля локализаций
    document.getElementById('title-en').value = '';
    document.getElementById('title-ru').value = '';
    document.getElementById('title-uk').value = '';
    document.getElementById('message-en').value = '';
    document.getElementById('message-ru').value = '';
    document.getElementById('message-uk').value = '';
    
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
    
    // Очищаем поля локализаций кнопки
    document.getElementById('button-text-en').value = '';
    document.getElementById('button-text-ru').value = '';
    document.getElementById('button-text-uk').value = '';
    
    // Очищаем URL и Callback кнопки
    document.getElementById('edit-button-url').value = '';
    document.getElementById('edit-button-callback').value = '';
    
    // Активный шаблон по умолчанию
    document.getElementById('template-active').checked = true;
}

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
