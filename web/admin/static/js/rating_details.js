document.addEventListener('DOMContentLoaded', function () {
    // Обработка кнопки распределения призов
    const distributePrizesBtn = document.getElementById('distributePrizesBtn');
    const distributeForm = document.getElementById('distributeForm');

    if (distributePrizesBtn && distributeForm) {
        distributePrizesBtn.addEventListener('click', function () {
            if (!confirm('Вы уверены, что хотите распределить призы? Это действие нельзя отменить.')) {
                return;
            }

            const formData = new FormData(distributeForm);
            const actionUrl = distributeForm.getAttribute('action');

            fetch(actionUrl, {
                    method: 'POST',
                    body: formData
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        alert('Призы успешно распределены!');
                        location.reload();
                    } else {
                        alert('Ошибка: ' + (data.error || 'Неизвестная ошибка'));
                    }
                })
                .catch(error => {
                    alert('Произошла ошибка при распределении призов');
                    console.error('Error:', error);
                });
        });
    }

    // Обработка кнопки отмены распределения призов
    const cancelPrizesBtn = document.getElementById('cancelPrizesBtn');
    const cancelPrizesForm = document.getElementById('cancelPrizesForm');

    if (cancelPrizesBtn && cancelPrizesForm) {
        cancelPrizesBtn.addEventListener('click', function () {
            if (!confirm('Вы уверены, что хотите отменить распределение призов? Это действие уменьшит баланс пользователей, которые получили призы.')) {
                return;
            }

            const actionUrl = cancelPrizesForm.getAttribute('action');

            fetch(actionUrl, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded'
                    },
                    body: new FormData(cancelPrizesForm)
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        alert('Распределение призов успешно отменено!');
                        location.reload();
                    } else {
                        alert('Ошибка: ' + (data.error || 'Неизвестная ошибка'));
                    }
                })
                .catch(error => {
                    alert('Произошла ошибка при отмене распределения призов');
                    console.error('Error:', error);
                });
        });
    }
});
