# 📘 Внутренний мануал: восстановление базы из бэкапа через pgAdmin

## 💻 1. Зайти в pgAdmin

Перейти в браузере по адресу:

```
http://localhost:8181/pgadmin
```

Используй заранее заданные учётные данные (логин/пароль).

---

## 💃 2. Очистить всю базу

1. В левой панели открыть нужную базу данных.
2. Кликнуть правой кнопкой по базе и выбрать `Query Tool`.
3. Вставить и выполнить следующий SQL-запрос:

```sql
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    ) LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE', r.schemaname, r.tablename);
    END LOOP;
END $$;
```

Этот скрипт удалит **все таблицы из всех схем кроме системных**.

---

## ♻️ 3. Восстановить базу из бэкапа

1. В той же базе нажать правой кнопкой → `Restore...`

2. В появившемся окне:

   * В поле **Filename** выбрать файл дампа по пути:

     ```
     /var/lib/pgadmin/storage/
     ```

     Это путь **внутри контейнера**.

   * Файл должен быть заранее положен на хосте в:

     ```
     /shared-data/pgadmin/storage
     ```

3. Нажать **Restore** и дождаться завершения.

---

✅ Готово! База восстановлена из дампа.
