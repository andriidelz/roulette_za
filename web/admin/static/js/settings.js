$(document).ready(function () {
    // Обработчик формы общих настроек
    $('#general-settings-form').submit(function (e) {
        e.preventDefault();
        saveSettings($(this));
    });

    // Обработчик формы настроек призового фонда
    $('#prize-settings-form').submit(function (e) {
        e.preventDefault();
        saveSettings($(this));
    });

    // Обработчик формы общих настроек
    $('#captcha-settings-form').submit(function (e) {
        e.preventDefault();
        saveSettings($(this));
    });

    // Функция для сохранения настроек
    function saveSettings(form) {
        // Собираем данные формы
        var formData = form.serialize();

        // Отправляем данные на сервер
        $.post('/admin/settings', formData, function (data) {
                if (data.success) {
                    toastr.success('Настройки успешно сохранены');
                } else {
                    toastr.error('Ошибка: ' + data.error);
                }
            })
            .fail(function (response) {
                toastr.error('Ошибка при сохранении настроек');
            });
    }
});
