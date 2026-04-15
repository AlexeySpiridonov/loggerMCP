# loggerMCP

MCP-сервер на Go для чтения и поиска по Ubuntu syslog.

## Возможности

- Чтение записей syslog с пагинацией
- Фильтрация по диапазону дат (начало/конец)
- Поиск по подстроке с поддержкой wildcards (`*`)
- Аутентификация по ключу доступа
- SSE-транспорт на порту 7777

## Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/AlexeySpiridonov/loggerMCP/main/install.sh | sudo bash
```

После установки:

```bash
# Задать ключ доступа
sudo nano /etc/loggermcp/config.yaml

# Запустить
sudo systemctl start loggermcp

# Проверить
sudo systemctl status loggermcp
```

## Настройка

```bash
cp config.yaml.example config.yaml
```

Отредактируйте `config.yaml`:

```yaml
access_key: "ваш-секретный-ключ"
syslog_path: "/var/log/syslog"
port: 7777
```

## Сборка и запуск

```bash
go build -o loggerMCP .
./loggerMCP
# или с указанием пути к конфигу:
./loggerMCP /path/to/config.yaml
```

## MCP Tool: `read_logs`

| Параметр     | Тип    | Обязательный | Описание                                                       |
|-------------|--------|-------------|----------------------------------------------------------------|
| access_key  | string | да          | Ключ доступа                                                   |
| start_date  | string | нет         | Начальная дата (`2006-01-02` или `2006-01-02T15:04:05`)        |
| end_date    | string | нет         | Конечная дата (`2006-01-02` или `2006-01-02T15:04:05`)         |
| pattern     | string | нет         | Фильтр по подстроке, `*` — wildcard. Пример: `error*disk`     |
| page        | number | нет         | Номер страницы (по умолчанию 1)                                |
| page_size   | number | нет         | Записей на странице (по умолчанию 100, макс 1000)              |

## TLS

В `config.yaml` включите `tls: true` — сервер автоматически сгенерирует self-signed сертификат при первом запуске. Или укажите свои:

```yaml
tls: true
cert_file: "/path/to/cert.pem"
key_file: "/path/to/key.pem"
```

## Установка через APT

### Добавить репозиторий

```bash
# Импортировать GPG-ключ
curl -fsSL https://AlexeySpiridonov.github.io/loggerMCP/public.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/loggermcp.gpg

# Добавить источник
echo "deb [signed-by=/usr/share/keyrings/loggermcp.gpg] https://AlexeySpiridonov.github.io/loggerMCP stable main" \
  | sudo tee /etc/apt/sources.list.d/loggermcp.list

# Установить
sudo apt update
sudo apt install loggermcp
```

После установки:
- Бинарник: `/usr/bin/loggermcp`
- Конфиг: `/etc/loggermcp/config.yaml` (создаётся из примера, **отредактируйте `access_key`**)
- Systemd-сервис: `loggermcp.service`

```bash
# Отредактировать конфиг
sudo nano /etc/loggermcp/config.yaml

# Запустить
sudo systemctl start loggermcp

# Проверить статус
sudo systemctl status loggermcp

# Логи сервиса
journalctl -u loggermcp -f
```

### Обновление

```bash
sudo apt update && sudo apt upgrade loggermcp
```

## Публикация нового релиза (деплой в APT)

### Одноразовая настройка

1. **GPG-ключ** для подписи репозитория:

```bash
gpg --full-generate-key
gpg --armor --export-secret-keys YOUR_KEY_ID
```

2. **GitHub Secrets** (Settings → Secrets and variables → Actions):
   - `GPG_PRIVATE_KEY` — приватный ключ (вывод команды выше)
   - `GPG_PASSPHRASE` — пароль ключа
   - `GPG_KEY_ID` — ID ключа

3. **GitHub Pages** — Settings → Pages → Source: `gh-pages` branch

### Релиз

Создайте тег — GitHub Actions автоматически соберёт `.deb` (amd64 + arm64), опубликует APT-репозиторий на GitHub Pages и создаст GitHub Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```
