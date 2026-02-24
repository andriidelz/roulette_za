window.deleteAdmin = function (id, username) {
  console.log("Delete triggered for:", id, username);
  if (!confirm(`Ви впевнені, що хочете видалити ${username}?`)) return;

  $.ajax({
    url: `/admin/admins/${id}/delete`,
    method: "POST",
    success: function () {
      console.log("Delete success");
      location.reload();
    },
    error: function (xhr) {
      console.error("Delete error:", xhr);
      alert(
        "Помилка при видаленні: " +
          (xhr.responseJSON?.error || "Unknown error"),
      );
    },
  });
};

window.toggleStatus = function (adminId, isActive) {
  console.log("Toggle status triggered for:", adminId, "New state:", isActive);
  const checkbox = $("#status_" + adminId);
  const originalState = !isActive;

  const formData = new FormData();
  formData.append("is_active", isActive ? "true" : "false");
  formData.append(
    "email",
    $(`tr[data-admin-id="${adminId}"]`).data("admin-email"),
  );

  $.ajax({
    url: `/admin/admins/${adminId}/update`,
    method: "POST",
    data: formData,
    processData: false,
    contentType: false,
    success: function () {
      console.log("Status updated successfully");
      location.reload();
    },
    error: function (xhr) {
      console.error("Toggle error:", xhr);
      checkbox.prop("checked", originalState);
      alert("Не вдалося змінити статус");
    },
  });
};

$(document).ready(function () {
  console.log("RBAC JS Loaded and Ready");

  $.ajaxSetup({
    headers: { "X-Requested-With": "XMLHttpRequest" },
  });

  $(document).on("click", ".edit-admin-btn", function (e) {
    e.preventDefault();
    const row = $(this).closest("tr");

    const adminId = row.data("admin-id");
    const email = row.data("admin-email");
    const firstName = row.data("admin-firstname");
    const lastName = row.data("admin-lastname");
    const isActive = row.attr("data-admin-active") === "true";
    const rolesAttr = String(row.attr("data-admin-roles") || "");
    const roleIds = rolesAttr.split(",").filter(Boolean).map(Number);

    $("#edit-admin-id").val(adminId);
    $("#edit-admin-email").val(email);
    $("#edit-admin-username").val(email);
    $("#edit-admin-firstname").val(firstName);
    $("#edit-admin-lastname").val(lastName);
    $("#edit-admin-active").prop("checked", isActive);

    $("#editAdminForm").attr("action", `/admin/admins/${adminId}/update`);

    $("#edit-admin-roles option").prop("selected", false);
    roleIds.forEach((id) => {
      $(`#edit-admin-roles option[value="${id}"]`).prop("selected", true);
    });

    const modalElement = document.getElementById("editAdminModal");
    const modal = bootstrap.Modal.getOrCreateInstance(modalElement);
    modal.show();
  });

  $("#editAdminForm").on("submit", function (e) {
    e.preventDefault();
    const formData = new FormData(this);
    const url = $(this).attr("action");

    $.ajax({
      url: url,
      method: "POST",
      data: formData,
      processData: false,
      contentType: false,
      success: function () {
        location.reload();
      },
      error: function (xhr) {
        alert("Помилка: " + (xhr.responseJSON?.error || "Unknown error"));
      },
    });
  });
});
