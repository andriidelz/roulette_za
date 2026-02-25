/**
 * RBAC Admin Users Management
 */

$(document).ready(function() {
    console.log('[RBAC] Script loaded');
    
    // ============= TABLE SORTING =============
    $('th').each(function(column) {
        $(this).click(function() {
            var table = $(this).parents('table').eq(0);
            var rows = table.find('tr:gt(0)').toArray().sort(comparer($(this).index()));
            this.asc = !this.asc;
            if (!this.asc) { rows = rows.reverse(); }
            for (var i = 0; i < rows.length; i++) { table.append(rows[i]); }
        });
    });

    function comparer(index) {
        return function(a, b) {
            var valA = getCellValue(a, index), valB = getCellValue(b, index);
            return $.isNumeric(valA) && $.isNumeric(valB) ? valA - valB : valA.toString().localeCompare(valB);
        };
    }

    function getCellValue(row, index) { 
        return $(row).children('td').eq(index).text(); 
    }

    // ============= EDIT ADMIN =============
    $(document).on('click', '.edit-admin-btn', function(e) {
        e.preventDefault();
        e.stopPropagation();
        
        const row = $(this).closest('tr');
        const adminId = row.data('admin-id');
        const email = row.data('admin-email');
        const firstName = row.data('admin-firstname');
        const lastName = row.data('admin-lastname');
        const isActive = row.data('admin-active') === true || row.data('admin-active') === 'true';
        const roleIds = (row.data('admin-roles') || '').toString().split(',').filter(Boolean).map(Number);

        console.log('[RBAC] Edit admin:', { adminId, email, firstName, lastName, isActive, roleIds });

        $('#edit-admin-id').val(adminId);
        $('#edit-admin-email').val(email);
        $('#edit-admin-firstname').val(firstName);
        $('#edit-admin-lastname').val(lastName);
        $('#edit-admin-active').prop('checked', isActive);

        $('#editAdminForm').attr('action', `/admin/rbac/users/${adminId}/update`);

        $('#edit-admin-roles option').prop('selected', false);
        roleIds.forEach((id) => {
            $(`#edit-admin-roles option[value="${id}"]`).prop('selected', true);
        });

        const modal = new bootstrap.Modal(document.getElementById('editAdminModal'));
        modal.show();
    });

    // ============= DELETE ADMIN =============
    $(document).on('click', '.delete-admin-btn', function(e) {
        e.preventDefault();
        e.stopPropagation();
        
        const adminId = $(this).data('id');
        const email = $(this).data('email') || "цього користувача";

        console.log('[RBAC] Delete admin triggered:', adminId, email);

        if (!confirm(`Ви впевнені, що хочете видалити адміністратора ${email}?`)) return;

        $.ajax({
            url: `/admin/rbac/users/${adminId}/delete`, 
            method: 'POST',
            headers: { 'X-Requested-With': 'XMLHttpRequest' },
            success: function(res) {
                console.log('[RBAC] Delete admin success:', res);
                showNotification('Адміністратора видалено', 'success');
                setTimeout(() => location.reload(), 1000);
            },
            error: function(xhr) {
                console.error('[RBAC] Delete admin error:', xhr);
                alert(xhr.responseJSON?.error || 'Помилка при видаленні адміністратора');
            }
        });
    });

    $(document).on('click', '.delete-role-btn', function(e) {
        e.preventDefault();
        e.stopPropagation();

        const roleId = $(this).data('id');
        const roleName = $(this).data('name') || "цю роль";

        console.log('[RBAC] Delete role triggered:', roleId, roleName);

        if (!confirm(`Ви впевнені, що хочете видалити роль "${roleName}"?`)) return;

        $.ajax({
            url: `/admin/rbac/roles/${roleId}/delete`,
            method: 'POST',
            headers: { 'X-Requested-With': 'XMLHttpRequest' },
            success: function(res) {
                console.log('[RBAC] Delete role success:', res);
                showNotification('Роль видалено', 'success');
                setTimeout(() => location.reload(), 1000);
            },
            error: function(xhr) {
                console.log('[RBAC] Delete role error:', roleId, roleName, xhr);
                alert(xhr.responseJSON?.error || 'Помилка при видаленні ролі');
            }
        });
    });

    // ============= TOGGLE STATUS (AJAX) =============
    $(document).on('change', '.status-toggle', function(e) {
        e.preventDefault();
        e.stopPropagation();
        
        const checkbox = $(this);
        const adminId = checkbox.data('admin-id');
        const isActive = checkbox.is(':checked');
        const originalState = !isActive;

        console.log('[RBAC] Status toggle:', { adminId, isActive });

        const formData = new FormData();
        formData.append('is_active', isActive ? 'true' : 'false');

        $.ajax({
            url: `/admin/rbac/users/${adminId}/update`,
            method: 'POST',
            data: formData,
            processData: false,
            contentType: false,
            headers: { 'X-Requested-With': 'XMLHttpRequest' },
            success: function(data) {
                console.log('[RBAC] Status success:', data);
                if (data.success) {
                    showNotification('Статус змінено успішно', 'success');
                } else {
                    checkbox.prop('checked', originalState);
                    showNotification(data.error || 'Помилка при зміні статусу', 'error');
                }
            },
            error: function(xhr) {
                checkbox.prop('checked', originalState);
                console.error('[RBAC] Status error:', xhr);
                const errorMsg = xhr.responseJSON?.error || 'Помилка з\'єднання';
                showNotification(errorMsg, 'error');
            }
        });
    });

    // ============= SUBMIT EDIT FORM =============
    $('#editAdminForm').on('submit', function(e) {
        e.preventDefault();
        
        const formData = new FormData(this);
        const url = $(this).attr('action');
        
        console.log('[RBAC] Submitting form:', url);

        $.ajax({
            url: url,
            method: 'POST',
            data: formData,
            processData: false,
            contentType: false,
            success: function(data) {
                console.log('[RBAC] Form success:', data);
                showNotification('Зміни збережено', 'success');
                setTimeout(() => location.reload(), 1000);
            },
            error: function(xhr) {
                console.error('[RBAC] Form error:', xhr);
                alert('Помилка: ' + (xhr.responseJSON?.error || 'Unknown error'));
            }
        });
    });

    // ============= NOTIFICATION HELPER =============
    function showNotification(message, type) {
        if (typeof window.showToast === 'function') {
            window.showToast(type, message);
        } else if (typeof showToast === 'function') {
            showToast(type, message);
        } else {
            console.log('[RBAC] Toast notification:', type, message);
            alert(message);
        }
    }

    window.showNotification = showNotification;

});

