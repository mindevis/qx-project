# Политика конфиденциальности

**Дата:** 2026-07-02  
**Контакт:** [mindevis.by@gmail.com](mailto:mindevis.by@gmail.com)

English summary: [PRIVACY.md on GitHub](https://github.com/mindevis/qx-project/blob/main/PRIVACY.md)

QXSystem — open-source экосистема Minecraft ([mc.qx-dev.ru](https://mc.qx-dev.ru)): QXWeb, QXApi, QXLauncher, QXAgent.

---

## Какие данные обрабатываются

| Данные | Где | Зачем |
| ------ | --- | ----- |
| Email и хэш пароля | QXApi | Регистрация и вход |
| JWT / refresh-токены | Браузер, QXLauncher | Сессия |
| ID устройства, версия лаунчера | QXLauncher ↔ API | Привязка QXLauncher к аккаунту |
| Токены Microsoft / Mojang (если привязали) | QXApi | Лицензионный запуск и подключение к серверам |
| Инстансы, моды, настройки серверов | БД QXApi | Функции платформы |
| IP, логи запросов | Nginx / API | Безопасность, rate limit |

Мы **не продаём** персональные данные третьим лицам.

---

## QXLauncher (Windows)

- Локально на ПК: токены, ID устройства, кэш (`%LOCALAPPDATA%\QXLauncher`).
- Сеть: API mc.qx-dev.ru, скачивание Minecraft/модов, обновление `qx-launcher.exe` по запросу пользователя.
- Самообновление заменяет только бинарник QX Launcher после явного действия (кнопка «Обновить» или авто-обновление из трея).

Подробнее: [Windows Defender и лаунчер](./launcher-windows-defender.md), [скачивание](./launcher-download.md).

---

## Хранение

Данные аккаунта — пока аккаунт существует. Логи сервера ротируются (обычно дни–недели).

---

## Ваши права

Запрос на доступ или удаление данных аккаунта: **mindevis.by@gmail.com**.

---

## Изменения

Актуальная версия всегда на этой странице. Краткая копия в репозитории: `PRIVACY.md`.

---

*См. также [security-legal.md](./security-legal.md).*
