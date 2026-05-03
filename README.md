# YstuPortal

## Web через Docker

### Требования
- Docker Desktop

### Запуск
Откройте web по HTTP через контейнер nginx, а API оставьте на `http://127.0.0.1:8080`.

```cmd
cd C:\Users\korob\OneDrive\Рабочий стол\проекты\YstuPortal
docker compose up -d
```

Откройте в браузере:
- http://127.0.0.1:8081/web_version.html

### Остановка
```cmd
docker compose down
```
