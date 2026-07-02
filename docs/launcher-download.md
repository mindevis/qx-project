# Скачивание QXLauncher (Windows)

Официальная страница загрузки в веб-панели: **[mc.qx-dev.ru/launcher](https://mc.qx-dev.ru/launcher)**

Прямая ссылка на файл (production):

**[https://mc.qx-dev.ru/downloads/qx-launcher.exe](https://mc.qx-dev.ru/downloads/qx-launcher.exe)**

---

## Code signing

Windows builds of QX Launcher are code-signed through the [SignPath Foundation](https://signpath.org/) open-source code signing program.

Сборки `qx-launcher.exe` для Windows подписываются через программу [SignPath Foundation](https://signpath.org/) для open-source проектов. Это снижает ложные срабатывания Windows Defender / SmartScreen.

Проверка подписи на ПК:

```powershell
Get-AuthenticodeSignature .\qx-launcher.exe | Format-List
```

Если подпись ещё не применена к конкретной сборке (переходный период), см. [Windows Defender](./launcher-windows-defender.md).

---

## Сборка из исходников

```bash
make build-launcher-win
# bin/qx-launcher.exe
```

См. [README](https://github.com/mindevis/qx-project/blob/main/README.md).

---

## Privacy

[Политика конфиденциальности](./privacy.md)
