$(document).ready(function () {
    // Бан пользователя
    $('.ban-user').click(function () {
        var userId = $(this).data('id');
        if (confirm('Вы уверены, что хотите забанить этого пользователя?')) {
            $.post('/admin/user/' + userId + '/ban', function (data) {
                if (data.success) {
                    window.location.reload();
                } else {
                    alert('Ошибка: ' + data.error);
                }
            });
        }
    });

    // Разбан пользователя
    $('.unban-user').click(function () {
        var userId = $(this).data('id');
        if (confirm('Вы уверены, что хотите разбанить этого пользователя?')) {
            $.post('/admin/user/' + userId + '/unban', function (data) {
                if (data.success) {
                    window.location.reload();
                } else {
                    alert('Ошибка: ' + data.error);
                }
            });
        }
    });

    // Быстрый просмотр пользователя
    $('.quick-view-btn').click(function () {
        var userId = $(this).data('id');
        $('#quickViewLoader').show();
        $('#quickViewContent').hide();
        $('#quickViewModal').modal('show');

        // Загрузка данных пользователя
        $.getJSON('/admin/user/' + userId + '/json', function (data) {
                // Заполняем основную информацию
                $('#qv-id').text(data.user.ID);
                $('#qv-telegram-id').text(data.user.TelegramID);
                $('#qv-username').html(data.user.Username ? '<a href="https://t.me/' + data.user.Username + '" target="_blank">@' + data.user.Username + '</a>' : '<span class="text-muted">—</span>');
                $('#qv-name').text((data.user.FirstName || '') + ' ' + (data.user.LastName || ''));
                $('#qv-created-at').text(new Date(data.user.CreatedAt).toLocaleString());
                $('#qv-wallet').text(data.user.WalletAddress || '—');

                // Заполняем статистику
                $('#qv-total-bets').text(data.stats.TotalBets || 0);
                $('#qv-won-bets').text(data.stats.WonBets || 0);
                $('#qv-efficiency').text(((data.stats.Efficiency || 0).toFixed(2)) + '%');
                $('#qv-total-points').text(data.stats.TotalPoints || 0);
                $('#qv-balance').text(data.user.Balance + ' $');

                if (data.rating) {
                    $('#qv-rating-position').text(data.rating.Position);
                } else {
                    $('#qv-rating-position').text('—');
                }

                // Заполняем таблицу ставок
                var betsHtml = '';
                if (data.bets && data.bets.length > 0) {
                    data.bets.forEach(function (bet) {
                        var betType = '';
                        if (bet.Option === 'red') {
                            betType = '<span class="badge bg-danger">Красное</span>';
                        } else if (bet.Option === 'black') {
                            betType = '<span class="badge bg-dark">Черное</span>';
                        } else {
                            betType = '<span class="badge bg-success">Зеро</span>';
                        }

                        var result = bet.Won ?
                            '<span class="text-success"><i class="bi bi-check-circle"></i> Выигрыш</span>' :
                            '<span class="text-danger"><i class="bi bi-x-circle"></i> Проигрыш</span>';

                        betsHtml += '<tr>' +
                            '<td>' + new Date(bet.CreatedAt).toLocaleString() + '</td>' +
                            '<td>' + betType + '</td>' +
                            '<td>' + result + '</td>' +
                            '<td>' + bet.Points + '</td>' +
                            '</tr>';
                    });
                } else {
                    betsHtml = '<tr><td colspan="4" class="text-center">Ставок не найдено</td></tr>';
                }
                $('#qv-bets-table').html(betsHtml);

                // Ссылка на полный профиль
                $('#qv-full-profile').attr('href', '/admin/user/' + data.user.ID);

                // Скрываем загрузчик и показываем содержимое
                $('#quickViewLoader').hide();
                $('#quickViewContent').show();
            })
            .fail(function () {
                alert('Ошибка при загрузке данных пользователя');
                $('#quickViewModal').modal('hide');
            });
    });

    // Быстрое редактирование пользователя
    $('.quick-edit-btn').click(function () {
        var userId = $(this).data('id');
        $('#quickEditLoader').show();
        $('#quickEditContent').hide();
        $('#quickEditModal').modal('show');

        // Загрузка данных пользователя
        $.getJSON('/admin/user/' + userId + '/json', function (data) {
                // Заполняем форму редактирования
                $('#edit-user-id').val(data.user.ID);
                $('#edit-username').val(data.user.Username);
                $('#edit-telegram-id').val(data.user.TelegramID);
                $('#edit-first-name').val(data.user.FirstName);
                $('#edit-last-name').val(data.user.LastName);
                $('#edit-language').val(data.user.LanguageCode);
                $('#edit-country').val(data.user.Country);
                $('#edit-wallet').val(data.user.WalletAddress);

                // Текущий баланс
                $('#current-balance strong').text(data.user.Balance + ' $');

                // Настройка статуса бана
                if (data.user.Banned) {
                    $('#ban-user-btn').hide();
                    $('#unban-user-btn').show();
                } else {
                    $('#ban-user-btn').show();
                    $('#unban-user-btn').hide();
                }

                // Устанавливаем обработчики для кнопок бана/разбана
                $('#ban-user-btn').off('click').on('click', function () {
                    if (confirm('Вы уверены, что хотите забанить этого пользователя?')) {
                        $.post('/admin/user/' + data.user.ID + '/ban', function (response) {
                            if (response.success) {
                                $('#ban-user-btn').hide();
                                $('#unban-user-btn').show();
                                alert('Пользователь успешно забанен');
                            } else {
                                alert('Ошибка: ' + response.error);
                            }
                        });
                    }
                });

                $('#unban-user-btn').off('click').on('click', function () {
                    if (confirm('Вы уверены, что хотите разбанить этого пользователя?')) {
                        $.post('/admin/user/' + data.user.ID + '/unban', function (response) {
                            if (response.success) {
                                $('#unban-user-btn').hide();
                                $('#ban-user-btn').show();
                                alert('Пользователь успешно разбанен');
                            } else {
                                alert('Ошибка: ' + response.error);
                            }
                        });
                    }
                });

                // Обработчик для изменения баланса
                $('#update-balance-btn').off('click').on('click', function () {
                    var amount = $('#balance-amount').val();
                    var operation = $('#balance-operation').val();

                    if (!amount || amount <= 0) {
                        alert('Пожалуйста, введите корректную сумму');
                        return;
                    }

                    $.post('/admin/user/' + data.user.ID + '/balance', {
                        amount: amount,
                        operation: operation
                    }, function (response) {
                        if (response.success) {
                            $('#current-balance strong').text(parseFloat(response.balance).toFixed(2) + ' $');
                            $('#balance-amount').val('');
                            alert('Баланс успешно обновлен');
                        } else {
                            alert('Ошибка: ' + response.error);
                        }
                    });
                });

                // Скрываем загрузчик и показываем содержимое
                $('#quickEditLoader').hide();
                $('#quickEditContent').show();
            })
            .fail(function () {
                alert('Ошибка при загрузке данных пользователя');
                $('#quickEditModal').modal('hide');
            });
    });

    // Сохранение изменений пользователя
    $('#save-user-btn').click(function () {
        var userId = $('#edit-user-id').val();
        var formData = {
            username: $('#edit-username').val(),
            firstName: $('#edit-first-name').val(),
            lastName: $('#edit-last-name').val(),
            languageCode: $('#edit-language').val(),
            country: $('#edit-country').val(),
            walletAddress: $('#edit-wallet').val()
        };

        $.post('/admin/user/' + userId + '/update', formData, function (data) {
            if (data.success) {
                alert('Данные пользователя успешно обновлены');
                $('#quickEditModal').modal('hide');
                // Обновляем строку пользователя в таблице
                var row = $('tr[data-id="' + userId + '"]');
                row.find('td:eq(2)').html(formData.username ? '<a href="https://t.me/' + formData.username + '" target="_blank">@' + formData.username + '</a>' : '<span class="text-muted">—</span>');
                row.find('td:eq(3)').text((formData.firstName || '') + ' ' + (formData.lastName || ''));

                // Обновляем язык
                var langHtml = '';
                if (formData.languageCode === 'en') {
                    langHtml = '<span class="badge bg-primary">English</span>';
                } else if (formData.languageCode === 'uk') {
                    langHtml = '<span class="badge bg-warning">Українська</span>';
                } else {
                    langHtml = '<span class="badge bg-secondary">' + formData.languageCode + '</span>';
                }
                row.find('td:eq(4)').html(langHtml);

                // Обновляем страну - для этого нужно будет перезагрузить страницу
                // так как у нас нет быстрого доступа к массиву стран в JS
                window.location.reload();
            } else {
                alert('Ошибка: ' + data.error);
            }
        });
    });

    // Экспорт списка пользователей в CSV
    $('#exportCSV').click(function (e) {
        e.preventDefault();
        exportTableToCSV('users_list.csv');
    });

    // Экспорт списка пользователей в Excel
    $('#exportExcel').click(function (e) {
        e.preventDefault();
        exportTableToExcel('usersTable', 'users_list');
    });

    // Функция экспорта таблицы в CSV
    function exportTableToCSV(filename) {
        var csv = [];
        var rows = document.querySelectorAll('#usersTable tr');

        for (var i = 0; i < rows.length; i++) {
            var row = [],
                cols = rows[i].querySelectorAll('td, th');

            for (var j = 0; j < cols.length - 1; j++) { // Исключаем колонку с действиями
                // Убираем HTML-теги и получаем только текст
                var text = cols[j].innerText;
                row.push('"' + text.replace(/"/g, '""') + '"');
            }

            csv.push(row.join(','));
        }

        // Скачиваем CSV-файл
        downloadCSV(csv.join('\n'), filename);
    }

    // Функция скачивания CSV-файла
    function downloadCSV(csv, filename) {
        var csvFile;
        var downloadLink;

        // Добавляем BOM для корректного отображения кириллицы в Excel
        var csv_with_bom = '\uFEFF' + csv;
        csvFile = new Blob([csv_with_bom], {
            type: 'text/csv;charset=utf-8;'
        });

        downloadLink = document.createElement('a');
        downloadLink.download = filename;
        downloadLink.href = window.URL.createObjectURL(csvFile);
        downloadLink.style.display = 'none';
        document.body.appendChild(downloadLink);

        downloadLink.click();
        document.body.removeChild(downloadLink);
    }

    // Функция экспорта таблицы в Excel
    function exportTableToExcel(tableID, filename = '') {
        var downloadLink;
        var dataType = 'application/vnd.ms-excel';
        var tableSelect = document.getElementById(tableID);
        var tableHTML = tableSelect.outerHTML.replace(/ /g, '%20');

        // Создаем временную таблицу, исключая колонку с действиями
        var tempTable = document.createElement('table');
        var headerRow = document.createElement('tr');

        // Копируем заголовки, кроме последнего (Действия)
        var headers = tableSelect.querySelectorAll('th');
        for (var i = 0; i < headers.length - 1; i++) {
            var th = document.createElement('th');
            th.innerText = headers[i].innerText;
            headerRow.appendChild(th);
        }
        tempTable.appendChild(headerRow);

        // Копируем данные, кроме колонки Действия
        var rows = tableSelect.querySelectorAll('tbody tr');
        for (var i = 0; i < rows.length; i++) {
            var row = document.createElement('tr');
            var cells = rows[i].querySelectorAll('td');
            for (var j = 0; j < cells.length - 1; j++) {
                var td = document.createElement('td');
                td.innerText = cells[j].innerText;
                row.appendChild(td);
            }
            tempTable.appendChild(row);
        }

        tableHTML = tempTable.outerHTML.replace(/ /g, '%20');

        // Укажем имя файла
        filename = filename ? filename + '.xls' : 'excel_data.xls';

        // Создаем ссылку для скачивания
        downloadLink = document.createElement('a');

        document.body.appendChild(downloadLink);

        // Добавляем BOM для корректного отображения кириллицы
        var bom = '\uFEFF';
        var dataUrl = 'data:' + dataType + ', ' + bom + tableHTML;
        downloadLink.href = dataUrl;
        downloadLink.download = filename;
        downloadLink.click();
        document.body.removeChild(downloadLink);
    }
});
