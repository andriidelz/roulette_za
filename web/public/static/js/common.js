/**
 * Общие JavaScript-функции для всех страниц
 */

// Функции, выполняющиеся после загрузки DOM
document.addEventListener('DOMContentLoaded', function() {
    setCurrentLanguage();

    // Инициализация всплывающих подсказок
    initTooltips();
    
    // Анимированный скролл до якорей
    initSmoothScroll();
    
    // Активация выпадающих меню
    initDropdowns();

});

function setCurrentLanguage() {
    const pathname = window.location.pathname;
    const currentLang = pathname.startsWith('/en/') ? 'en' : pathname.startsWith('/ru/') ? 'ru' : 'uk';

    const languages = document.querySelectorAll('[data-lang]');
    const currentFlag = document.querySelector('.lang-flag');

    languages.forEach((lang) => {
        if (lang.getAttribute('data-lang') === currentLang) {
            const flag = lang.querySelector('.fi');
            flag.classList.forEach((className) => {
                currentFlag.classList.add(className);
            })
            lang.classList.add('d-none');
        }        
    });
}

/**
 * Инициализация всплывающих подсказок Bootstrap
 */
function initTooltips() {
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
}

/**
 * Плавный скролл к якорям на странице
 */
function initSmoothScroll() {
    document.querySelectorAll('a[href^="#"]:not([data-bs-toggle])').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            
            const targetId = this.getAttribute('href');
            if (targetId === '#') return;
            
            const targetElement = document.querySelector(targetId);
            
            if (targetElement) {
                window.scrollTo({
                    top: targetElement.offsetTop - 70,
                    behavior: 'smooth'
                });
            }
        });
    });
}

/**
 * Инициализация выпадающих меню
 */
function initDropdowns() {
    const dropdownElementList = [].slice.call(document.querySelectorAll('.dropdown-toggle'));
    dropdownElementList.map(function (dropdownToggleEl) {
        return new bootstrap.Dropdown(dropdownToggleEl);
    });
}

/**
 * Копирование текста в буфер обмена
 * @param {string} text - Текст для копирования
 * @param {function} callback - Функция обратного вызова после копирования
 */
function copyToClipboard(text, callback) {
    navigator.clipboard.writeText(text)
        .then(() => {
            if (callback && typeof callback === 'function') {
                callback(true);
            }
        })
        .catch(err => {
            console.error('Ошибка при копировании: ', err);
            if (callback && typeof callback === 'function') {
                callback(false, err);
            }
        });
}

/**
 * Показать уведомление
 * @param {string} message - Сообщение для отображения
 * @param {string} type - Тип уведомления (success, error, info, warning)
 * @param {number} duration - Длительность отображения в миллисекундах
 */
function showNotification(message, type = 'info', duration = 3000) {
    // Проверяем, существует ли контейнер для уведомлений
    let notificationContainer = document.getElementById('notification-container');
    
    // Если нет, создаем его
    if (!notificationContainer) {
        notificationContainer = document.createElement('div');
        notificationContainer.id = 'notification-container';
        notificationContainer.style.position = 'fixed';
        notificationContainer.style.top = '20px';
        notificationContainer.style.right = '20px';
        notificationContainer.style.zIndex = '9999';
        document.body.appendChild(notificationContainer);
    }
    
    // Определяем класс в зависимости от типа
    let className;
    switch (type) {
        case 'success':
            className = 'alert-success';
            break;
        case 'error':
            className = 'alert-danger';
            break;
        case 'warning':
            className = 'alert-warning';
            break;
        case 'info':
        default:
            className = 'alert-info';
            break;
    }
    
    // Создаем уведомление
    const notification = document.createElement('div');
    notification.className = `alert ${className} alert-dismissible fade show`;
    notification.style.marginBottom = '10px';
    notification.style.minWidth = '250px';
    notification.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;
    
    // Добавляем уведомление в контейнер
    notificationContainer.appendChild(notification);
    
    // Автоматически закрываем уведомление через указанное время
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => {
            notificationContainer.removeChild(notification);
        }, 300);
    }, duration);
}
