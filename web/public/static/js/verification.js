/**
 * JavaScript для страницы проверки результатов рулетки
 */

document.addEventListener('DOMContentLoaded', function() {
    // Инициализация обработчиков формы и кнопок
    initVerificationForm();
    initVerifyButtons();
    initModalVerification();
    initCopyButtons();
    
    // Подсветка и прокрутка к строке с ID, если есть
    highlightAndScrollToRow();
});

/**
 * Инициализация формы проверки хеша
 */
function initVerificationForm() {
    const verifyForm = document.getElementById('verify-form');
    if (!verifyForm) return;
    
    verifyForm.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const number = document.getElementById('verify-number').value;
        const salt = document.getElementById('verify-salt').value;
        const originalHash = document.getElementById('verify-hash').value;
        
        if (!number || !salt || !originalHash) {
            showNotification('Пожалуйста, заполните все поля', 'warning');
            return;
        }
        
        verifyHash(number, salt, originalHash);
    });
}

/**
 * Инициализация кнопок проверки в таблице
 */
function initVerifyButtons() {
    const verifyButtons = document.querySelectorAll('.verify-hash-btn');
    if (!verifyButtons.length) return;
    
    verifyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const number = this.dataset.number;
            const hash = this.dataset.hash;
            const salt = this.dataset.salt;
            const base62 = this.dataset.base62;
            
            // Заполняем модальное окно проверки
            document.getElementById('modal-verify-number').value = number;
            document.getElementById('modal-verify-salt').value = salt;
            document.getElementById('modal-verify-hash').value = hash;
            document.getElementById('modal-verify-computed-hash').value = '';
            
            // Сбрасываем сообщение о результате
            const resultElement = document.getElementById('modal-verify-result');
            resultElement.className = 'alert alert-info';
            resultElement.innerHTML = '<i class="bi bi-info-circle me-2"></i>Нажмите "Проверить", чтобы подтвердить подлинность результата ID: ' + base62;
            
            // Показываем модальное окно проверки
            const verifyModal = new bootstrap.Modal(document.getElementById('verifyHashModal'));
            verifyModal.show();
        });
    });
}

/**
 * Инициализация кнопки проверки в модальном окне
 */
function initModalVerification() {
    const modalVerifyButton = document.getElementById('modal-do-verify-hash');
    if (!modalVerifyButton) return;
    
    modalVerifyButton.addEventListener('click', function() {
        const number = document.getElementById('modal-verify-number').value;
        const salt = document.getElementById('modal-verify-salt').value;
        const originalHash = document.getElementById('modal-verify-hash').value;
        
        verifyHash(number, salt, originalHash);
    });
}

/**
 * Функция проверки хеша
 * @param {string|number} number - Число (0-36)
 * @param {string} salt - Криптографическая соль
 * @param {string} originalHash - Оригинальный хеш для сравнения
 */
function verifyHash(number, salt, originalHash) {
    // Отправляем запрос на проверку хеша
    fetch('/api/verify-hash', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            number: parseInt(number),
            salt: salt,
            originalHash: originalHash
        })
    })
    .then(response => response.json())
    .then(data => {
        // Обновляем поля с вычисленным хешем
        if (document.getElementById('verify-computed-hash')) {
            document.getElementById('verify-computed-hash').value = data.computedHash;
        }
        if (document.getElementById('modal-verify-computed-hash')) {
            document.getElementById('modal-verify-computed-hash').value = data.computedHash;
        }
        if (document.getElementById('computed-hash-text')) {
            document.getElementById('computed-hash-text').textContent = data.computedHash;
        }
        
        // Обновляем результат проверки в форме на странице
        const verifyResult = document.getElementById('verify-result');
        if (verifyResult) {
            verifyResult.classList.remove('d-none', 'alert-info', 'alert-danger', 'alert-success');
            
            if (data.valid) {
                verifyResult.classList.add('alert-success');
                document.getElementById('verify-message').innerHTML = '<i class="bi bi-check-circle me-2"></i><strong>Результат подтвержден!</strong> Данные подлинны, результат не был изменен.';
            } else {
                verifyResult.classList.add('alert-danger');
                document.getElementById('verify-message').innerHTML = '<i class="bi bi-x-circle me-2"></i><strong>Результат недействителен!</strong> Данные могли быть изменены.';
            }
        }
        
        // Обновляем результат проверки в модальном окне
        const modalVerifyResult = document.getElementById('modal-verify-result');
        if (modalVerifyResult) {
            modalVerifyResult.classList.remove('alert-info', 'alert-danger', 'alert-success');
            
            if (data.valid) {
                modalVerifyResult.classList.add('alert-success');
                modalVerifyResult.innerHTML = '<i class="bi bi-check-circle me-2"></i><strong>Результат подтвержден!</strong> Данные подлинны, результат не был изменен.';
            } else {
                modalVerifyResult.classList.add('alert-danger');
                modalVerifyResult.innerHTML = '<i class="bi bi-x-circle me-2"></i><strong>Результат недействителен!</strong> Данные могли быть изменены.';
            }
        }
    })
    .catch(error => {
        console.error('Ошибка при проверке хеша:', error);
        
        // Отображаем ошибку в форме и модальном окне
        if (document.getElementById('verify-result')) {
            document.getElementById('verify-result').classList.remove('d-none', 'alert-info', 'alert-success').classList.add('alert-danger');
            document.getElementById('verify-message').innerHTML = '<i class="bi bi-exclamation-triangle me-2"></i><strong>Ошибка!</strong> Не удалось выполнить проверку. Попробуйте позже.';
        }
        
        if (document.getElementById('modal-verify-result')) {
            document.getElementById('modal-verify-result').classList.remove('alert-info', 'alert-success').classList.add('alert-danger');
            document.getElementById('modal-verify-result').innerHTML = '<i class="bi bi-exclamation-triangle me-2"></i><strong>Ошибка!</strong> Не удалось выполнить проверку. Попробуйте позже.';
        }
        
        showNotification('Ошибка проверки: ' + error.message, 'error');
    });
}

/**
 * Инициализация кнопок копирования кода
 */
function initCopyButtons() {
    const copyButtons = document.querySelectorAll('.copy-btn');
    if (!copyButtons.length) return;
    
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const codeBlock = this.closest('.code-block, .code-block-dark');
            const codeText = codeBlock.querySelector('pre code').textContent;
            
            copyToClipboard(codeText, (success) => {
                if (success) {
                    this.textContent = 'Скопировано!';
                    setTimeout(() => {
                        this.textContent = 'Копировать';
                    }, 2000);
                    showNotification('Код успешно скопирован!', 'success');
                } else {
                    this.textContent = 'Ошибка';
                    setTimeout(() => {
                        this.textContent = 'Копировать';
                    }, 2000);
                    showNotification('Не удалось скопировать код.', 'error');
                }
            });
        });
    });
}

/**
 * Подсветка и прокрутка к строке таблицы, если есть ID для подсветки
 */
function highlightAndScrollToRow() {
    const highlightedRow = document.querySelector('.highlight-row');
    if (!highlightedRow) return;
    
    // Прокручиваем к подсвеченной строке с небольшой задержкой
    setTimeout(() => {
        highlightedRow.scrollIntoView({
            behavior: 'smooth',
            block: 'center'
        });
    }, 500);
}
