// Создаем уникальный токен для предотвращения повторной отправки
var actionTokens = {};

$(document).ready(function () {
    // Инициализация тултипов Bootstrap
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'))
    var tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl)
    });

    let filter = { period: 'day', dateFrom: '', dateTo: '' };

    // Обработчики фильтров периода
    document.querySelectorAll('[data-period]').forEach(button => {
        button.addEventListener('click', function () {
            document.querySelectorAll('[data-period]').forEach(btn => {
                btn.classList.remove('active');
            });
            this.classList.add('active');

            filter.dateFrom = '';
            filter.dateTo = '';
            filter.period = this.getAttribute('data-period');

            $('#start-date').val('');
            $('#end-date').val('');
            update();
        });
    });

    $('#apply-date-filter').click(function () {
        $('#loading').show();

        filter.dateFrom = $('#start-date').val();
        filter.dateTo = $('#end-date').val();
        filter.period = '';
        update();
    });

    function update() {
        const params = new URLSearchParams();

        params.append('dateFrom', filter.dateFrom);
        params.append('dateTo', filter.dateTo);
        params.append('period', filter.period);


        $.get('/admin/withdrawals/stat?' + params.toString(), function (result) {
            $('#loading').hide();

            const tbody = $('#statTable');
            tbody.empty();

            if (result.length === 0) {
                tbody.append('<tr><td colspan="10" class="text-center text-muted">Статистика за период не найдена</td></tr>');
                return;
            }

            result.forEach(data => {
                let row = `
        <tr>
          <td><strong>${data.Day}</strong></td>
          <td>${data.Earn}</td>
          <td>${data.Withdrawal}</td>
          <td>${data.Payout}</td>
          <td>${data.Balance}</td>
        </tr>
      `;
                tbody.append(row);
            });
        }).fail(function () {
            $('#loading').hide();
            alert('Не удалось загрузить статистику');
        });
    };

    update();

    // Очищаем все ранее установленные обработчики
    $('.approve-withdrawal, .reject-withdrawal').off();

    // Функция для копирования текста в буфер обмена
    $('.copy-text-btn').click(function () {
        var text = $(this).data('clipboard-text');
        navigator.clipboard.writeText(text).then(function () {
            // Показываем уведомление об успешном копировании
            var tooltip = bootstrap.Tooltip.getInstance(this) ||
                new bootstrap.Tooltip(this, {
                    title: 'Скопировано!',
                    trigger: 'manual'
                });

            $(this).attr('data-bs-original-title', 'Скопировано!');
            tooltip.show();

            setTimeout(function () {
                tooltip.hide();
                $(this).attr('data-bs-original-title', 'Копировать');
            }.bind(this), 1000);
        }.bind(this)).catch(function (error) {
            console.error('Ошибка при копировании: ', error);
        });
    });

    // Создаем уникальные токены для каждой кнопки действия
    $('.approve-withdrawal, .reject-withdrawal').each(function () {
        var id = $(this).data('id');
        var action = $(this).hasClass('approve-withdrawal') ? 'approve' : 'reject';
        var key = action + '-' + id;
        actionTokens[key] = Math.random().toString(36).substring(2, 15);
    });

    // Обработчик подтверждения вывода
    $('.approve-withdrawal').click(function (e) {
        e.preventDefault();
        e.stopPropagation();

        var withdrawalId = $(this).data('id');
        var btn = $(this);
        var tokenKey = 'approve-' + withdrawalId;

        // console.log('Клик по кнопке подтверждения ID:', withdrawalId);

        // Проверяем, существует ли токен (если нет, значит действие уже выполняется)
        if (!actionTokens[tokenKey]) {
            console.log('Токен отсутствует, запрос уже в процессе обработки');
            return false;
        }

        if (confirm('Вы уверены, что хотите подтвердить этот вывод? Это действие невозможно отменить.')) {
            // Сохраняем токен и удаляем его из списка
            var token = actionTokens[tokenKey];
            delete actionTokens[tokenKey];
            
            if (withdrawalId === 'all') {
                // для всіх виплат видаляємо всі токени
                actionTokens = {};
            }

            // Блокируем кнопку и показываем индикатор загрузки
            btn.prop('disabled', true).html('<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Обработка...');

            // Используем AJAX вместо $.post для дополнительных настроек
            $.ajax({
                url: '/admin/withdrawal/' + withdrawalId + '/approve',
                type: 'POST',
                headers: {
                    'X-Action-Token': token,  // Добавляем токен в заголовок
                    'X-Requested-With': 'XMLHttpRequest'
                },
                cache: false,
                success: function (data) {
                    if (data.success) {
                        window.location.reload();
                    } else {
                        alert('Ошибка: ' + data.error);
                        // Восстанавливаем токен и возвращаем кнопку в исходное состояние
                        actionTokens[tokenKey] = token;
                        btn.prop('disabled', false).html('<i class="bi bi-check-circle me-1"></i>Подтвердить');
                    }
                },
                error: function (xhr) {
                    console.error('Ошибка запроса:', xhr.responseText);
                    alert('Произошла ошибка при обработке запроса. Проверьте консоль для деталей.');
                    // Восстанавливаем токен и возвращаем кнопку в исходное состояние
                    actionTokens[tokenKey] = token;
                    btn.prop('disabled', false).html('<i class="bi bi-check-circle me-1"></i>Подтвердить');
                }
            });
        }
    });

    // Обработчик отклонения вывода
    $('.reject-withdrawal').click(function (e) {
        e.preventDefault();
        e.stopPropagation();

        var withdrawalId = $(this).data('id');
        var btn = $(this);
        var tokenKey = 'reject-' + withdrawalId;

        // console.log('Клик по кнопке отклонения ID:', withdrawalId);

        // Проверяем, существует ли токен (если нет, значит действие уже выполняется)
        if (!actionTokens[tokenKey]) {
            console.log('Токен отсутствует, запрос уже в процессе обработки');
            return false;
        }

        if (confirm('Вы уверены, что хотите отклонить этот вывод? Средства будут возвращены на баланс пользователя.')) {
            // Сохраняем токен и удаляем его из списка
            var token = actionTokens[tokenKey];
            delete actionTokens[tokenKey];

            // Блокируем кнопку и показываем индикатор загрузки
            btn.prop('disabled', true).html('<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Обработка...');

            // Используем AJAX вместо $.post для дополнительных настроек
            $.ajax({
                url: '/admin/withdrawal/' + withdrawalId + '/reject',
                type: 'POST',
                headers: {
                    'X-Action-Token': token,  // Добавляем токен в заголовок
                    'X-Requested-With': 'XMLHttpRequest'
                },
                cache: false,
                success: function (data) {
                    if (data.success) {
                        window.location.reload();
                    } else {
                        alert('Ошибка: ' + data.error);
                        // Восстанавливаем токен и возвращаем кнопку в исходное состояние
                        actionTokens[tokenKey] = token;
                        btn.prop('disabled', false).html('<i class="bi bi-x-circle me-1"></i>Отклонить');
                    }
                },
                error: function (xhr) {
                    console.error('Ошибка запроса:', xhr.responseText);
                    alert('Произошла ошибка при обработке запроса. Проверьте консоль для деталей.');
                    // Восстанавливаем токен и возвращаем кнопку в исходное состояние
                    actionTokens[tokenKey] = token;
                    btn.prop('disabled', false).html('<i class="bi bi-x-circle me-1"></i>Отклонить');
                }
            });
        }
    });

    // Обработчик клика на кнопку информации для отображения модального окна с деталями
    $('[data-bs-toggle="tooltip"][title="Показать детали"]').click(function () {
        var row = $(this).closest('tr');

        // Получаем данные из строки таблицы
        var id = row.find('td:eq(0)').text().trim();
        var user = row.find('td:eq(1)').text().trim();
        var amount = row.find('td:eq(2)').text().trim();
        var wallet = row.find('td:eq(3) .hash-label').text().trim();
        var date = row.find('td:eq(4)').text().trim();
        var status = row.find('td:eq(5) .badge').text().trim();
        var transaction = row.find('td:eq(6) .hash-label').text().trim();
        var providerName = transaction ? 'OxaPay' : 'Не указан';

        // Заполняем модальное окно
        $('#modal-id').text(id);
        $('#modal-user').text(user);
        $('#modal-amount').text(amount);
        $('#modal-wallet').text(wallet);
        $('#modal-copy-wallet').data('clipboard-text', wallet);
        $('#modal-created').text(date);
        $('#modal-updated').text(date); // В таблице нет обновления, поэтому используем ту же дату
        $('#modal-status').text(status);
        $('#modal-provider').text(providerName);
        $('#modal-transaction').text(transaction || 'Нет данных');
        $('#modal-copy-transaction').data('clipboard-text', transaction);

        // Если есть хеш транзакции и провайдер OxaPay, добавляем ссылку на блокчейн-обозреватель
        if (transaction && providerName === 'OxaPay') {
            $('#modal-blockchain-link').removeClass('d-none');
            $('#modal-blockchain-link a').attr('href', 'https://tronscan.org/#/transaction/' + transaction);
        } else {
            $('#modal-blockchain-link').addClass('d-none');
        }

        // Открываем модальное окно
        var modal = new bootstrap.Modal(document.getElementById('withdrawalDetailsModal'));
        modal.show();
    });

    // Обработчик обновления списка запросов на вывод
    $('.refresh-withdrawals').click(function () {
        // Добавляем класс для анимации вращения
        $(this).addClass('spinning');

        // Перезагружаем страницу
        window.location.reload();
    });

    // Обработчик обновления истории выводов
    $('.refresh-history').click(function () {
        // Добавляем анимацию вращения иконки
        const icon = $(this).find('i');
        icon.addClass('spinning');

        // Перезагружаем страницу для получения актуальных данных
        window.location.reload();
    });

    // Обработчик изменения фильтра статусов историй выводов
    $('#historyStatusFilter').change(function () {
        const selectedStatus = $(this).val();

        if (selectedStatus) {
            // Фильтруем строки таблицы истории
            $('.withdrawal-history-item').each(function () {
                const rowStatus = $(this).data('status');
                if (rowStatus === selectedStatus) {
                    $(this).show();
                } else {
                    $(this).hide();
                }
            });
        } else {
            // Если выбраны "Все статусы", показываем все строки
            $('.withdrawal-history-item').show();
        }

        // Проверяем, отображаются ли какие-либо строки
        const visibleRows = $('.withdrawal-history-item:visible').length;

        if (visibleRows === 0) {
            // Если нет отображаемых строк, показываем сообщение
            if ($('#no-history-data').length === 0) {
                $('#withdrawalHistoryTable tbody').append(
                    '<tr id="no-history-data"><td colspan="8" class="text-center">' +
                    '<div class="alert alert-info mb-0">' +
                    '<i class="bi bi-info-circle me-2"></i>Нет данных для выбранного статуса' +
                    '</div></td></tr>'
                );
            }
        } else {
            // Если есть отображаемые строки, удаляем сообщение
            $('#no-history-data').remove();
        }
    });

});
