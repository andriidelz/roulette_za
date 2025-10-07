/**
 * JavaScript для страницы примера проверки хеша
 */

document.addEventListener('DOMContentLoaded', function() {
    // Инициализация формы проверки
    initVerificationForm();
    
    // Инициализация кнопки копирования кода
    initCopyButton();
});

/**
 * Инициализация формы проверки хеша
 */
function initVerificationForm() {
    const verificationForm = document.getElementById('verification-form');
    if (!verificationForm) return;
    
    verificationForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Получаем значения из формы
        const number = parseInt(document.getElementById('number-input').value);
        const salt = document.getElementById('salt-input').value;
        const originalHash = document.getElementById('hash-input').value.trim();
        
        // Показываем карточку с результатом
        document.getElementById('result-card').classList.remove('d-none');
        
        // Обновляем блок результата
        const resultBlock = document.getElementById('verification-result');
        resultBlock.innerHTML = '<i class="bi bi-hourglass-split me-2"></i> Проверка...';
        resultBlock.className = 'alert alert-info';
        
        // Проверяем хеш
        try {
            const result = await verifyHash(number, salt, originalHash);
            
            // Обновляем данные в интерфейсе
            document.getElementById('input-data').textContent = result.input;
            document.getElementById('computed-hash').textContent = result.computedHash;
            
            if (originalHash) {
                document.getElementById('original-hash').textContent = result.originalHash;
                document.getElementById('original-hash-block').classList.remove('d-none');
            } else {
                document.getElementById('original-hash-block').classList.add('d-none');
            }
            
            // Обновляем блок результата в зависимости от результата проверки
            if (result.error) {
                resultBlock.innerHTML = `<i class="bi bi-exclamation-triangle me-2"></i> Ошибка: ${result.error}`;
                resultBlock.className = 'alert alert-danger';
            } else if (!originalHash) {
                resultBlock.innerHTML = '<i class="bi bi-info-circle me-2"></i> Хеш успешно вычислен! Скопируйте его для использования в проверке.';
                resultBlock.className = 'alert alert-success';
                
                // Заполняем поле с оригинальным хешем
                document.getElementById('hash-input').value = result.computedHash;
            } else if (result.isValid) {
                resultBlock.innerHTML = '<i class="bi bi-check-circle me-2"></i> Хеш подтвержден! Данные подлинны.';
                resultBlock.className = 'alert alert-success';
            } else {
                resultBlock.innerHTML = '<i class="bi bi-x-circle me-2"></i> Хеш не совпадает! Данные могли быть изменены.';
                resultBlock.className = 'alert alert-danger';
            }
        } catch (error) {
            resultBlock.innerHTML = `<i class="bi bi-exclamation-triangle me-2"></i> Ошибка: ${error.message}`;
            resultBlock.className = 'alert alert-danger';
        }
    });
}

/**
 * Инициализация кнопки копирования кода
 */
function initCopyButton() {
    const translations = {
        en: {
            copySuccess: 'Code copied successfully!',
            copyError: 'Failed to copy code.',
            copied: 'Copied!',
            copy: 'Copy',
        },
        uk: {
            copySuccess: 'Код успішно скопійований!',
            copyError: 'Не вдалося скопіювати код.',
            copied: 'Скопійовано!',
            copy: 'Копіювати',
        },
        ru: {
            copySuccess: 'Код успешно скопирован!',
            copyError: 'Не удалось скопировать код.',
            copied: 'Скопировано!',
            copy: 'Копировать',
        }
    };

    const copyBtn = document.getElementById('copy-code-btn');
    if (!copyBtn) return;

    const lang = document.documentElement.lang || 'en'; 
    const text = translations[lang] || translations['en'];
    
    copyBtn.addEventListener('click', function() {
        const codeText = document.querySelector('.code-block-dark pre code').textContent;
        
        copyToClipboard(codeText, (success) => {
            if (success) {
                this.textContent = text.copied;
                setTimeout(() => {
                    this.textContent = text.copy;
                }, 2000);
                showNotification(text.copySuccess, 'success');
            } else {
                this.textContent = 'Ошибка';
                setTimeout(() => {
                    this.textContent = text.copy;
                }, 2000);
                showNotification(text.copyError, 'error');
            }
        });
    });
}

/**
 * Функция для проверки хеша SHA-256
 * @param {number} number - Число (0-36)
 * @param {string} salt - Криптографическая соль
 * @param {string} originalHash - Оригинальный хеш для сравнения (опционально)
 * @returns {Promise<Object>} - Результат проверки
 */
async function verifyHash(number, salt, originalHash) {
    try {
        // Формируем строку в формате "число:соль"
        const data = `${number}:${salt}`;
        
        // Кодируем строку в массив байтов
        const encoder = new TextEncoder();
        const dataBuffer = encoder.encode(data);
        
        // Вычисляем SHA-256 хеш
        const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
        
        // Преобразуем ArrayBuffer в строку hex
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const computedHash = hashArray.map(b => 
            b.toString(16).padStart(2, '0')
        ).join('');
        
        // Проверяем, совпадает ли вычисленный хеш с оригинальным
        const isValid = !originalHash || computedHash === originalHash;
        
        return {
            isValid,
            computedHash,
            originalHash: originalHash || computedHash,
            number,
            salt,
            input: data
        };
    } catch (error) {
        console.error('Ошибка при проверке хеша:', error);
        return {
            isValid: false,
            error: error.message
        };
    }
}
