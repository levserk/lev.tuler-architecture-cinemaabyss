# Ответ на Задание 1

**Контейнерная диаграмма C4:** [CinemaAbyss_ToBe_Architecture.puml](architecture/CinemaAbyss_ToBe_Architecture.puml)

![CinemaAbyss To-Be Architecture](architecture/CinemaAbyss_ToBe_Architecture.png)

Разделил систему на 4 домена:

- Users - пользователи и профили
- Movies - каталог фильмов
- Payments - платежи
- Subscriptions - подписки

Каждый домен имеет свой сервис и БД. Все запросы идут через API Gateway. Для асинхронного взаимодействия используется Kafka. Внешние системы: рекомендации и платежный шлюз.

# Ответ на Задание 2

## Часть 1. Реализация прокси-сервиса

**Proxy Service (Strangler Fig pattern):**

- API Gateway на порту 8000 с маршрутизацией запросов
- Постепенная миграция трафика через `MOVIES_MIGRATION_PERCENT` (50% по умолчанию)
- Прозрачное проксирование к monolith и movies-service
- Health check endpoint: `/health`

**Тестирование:**

```bash
curl http://localhost:8000/api/movies  # Работает через proxy
```

## Часть 2. Реализация Kafka

**Events Service (MVP с Producer/Consumer):**

- Реальная интеграция с Kafka (библиотека `github.com/segmentio/kafka-go`)
- Producer/Consumer для топиков: `movie-events`, `user-events`, `payment-events`
- API endpoints:
  - `/api/events/health` - проверка здоровья
  - `/api/events/movie` - создание событий фильмов
  - `/api/events/user` - создание пользовательских событий
  - `/api/events/payment` - создание событий платежей
- Автосоздание топиков и consumer groups

**Результаты тестирования:**

![Postman тесты](screenshots/task2-test.png)

**Все тесты зеленые:** 22/22 запроса успешно, 42/42 утверждения прошли

**Kafka топики с сообщениями:**

![Kafka UI топики](screenshots/task2-kafka.png)

Топики созданы автоматически

**Архитектурные решения:**

- Использованы Bitnami образы Kafka/ZooKeeper для кроссплатформенности
- Реализован полный цикл: API → Producer → Kafka → Consumer → Логирование
- Events service сам создает и читает сообщения (MVP требование)

# Ответ на Задание 3

## Часть 1. Настройка CI/CD

**Доработка GitHub Actions:**

- Добавил сборку `proxy` и `events` сервисов в `.github/workflows/docker-build-push.yml`
- Настроил push образов в GitHub Container Registry
- Все workflow проходят успешно 
- Образы доступны в registry для Kubernetes деплоя

## Часть 2. Настройка Kubernetes

**Созданные манифесты:**

1. **`events-service.yaml`** - Deployment + Service для events сервиса

   - Образ: `ghcr.io/username/events-service:latest`
   - Порт: 8082
   - Переменные: `KAFKA_BROKERS`, `PORT`

2. **`proxy-service.yaml`** - Deployment + Service для proxy сервиса

   - Образ: `ghcr.io/username/proxy-service:latest`
   - Порт: 8000
   - Переменные: `MOVIES_MIGRATION_PERCENT`, URLs сервисов

3. **`ingress.yaml`** - маршрутизация трафика
   - `cinemaabyss.example.com/` → proxy-service:8000
   - `cinemaabyss.example.com/api/events` → events-service:8082

**Пошаговый деплой:**

```bash
# 1. Создание namespace
kubectl create namespace cinemaabyss

# 2. Применение манифестов
kubectl apply -f src/kubernetes/

# 3. Проверка статуса
kubectl get pods -n cinemaabyss
```

**Результаты деплоя:**

- Все 7 подов запущены и работают
- API доступен: `https://cinemaabyss.example.com/api/movies`
- Events API работает: `https://cinemaabyss.example.com/api/events/health`
- Newman тесты: 22/22 успешны (events тесты проходят)

**Скриншоты:**
![API вызов](screenshots/curl.png)
![Events логи](screenshots/events-service%20logs.png)

# Ответ на Задание 4

## Реализация Helm-чартов

**Настройка values.yaml:**

- Образы всех сервисов: `ghcr.io/levserk/lev.tuler-architecture-cinemaabyss/[service]:latest`
- Конфигурация ресурсов для всех компонентов
- Настройка Kafka/ZooKeeper с Bitnami образами
- Переменные для Strangler Fig: `gradualMigration: "true"`, `moviesMigrationPercent: "100"`

**Структура Helm chart:**

```
helm/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── configmap.yaml          # Конфигурация приложения
    ├── secret.yaml             # Пароли и секреты
    ├── ingress.yaml            # Маршрутизация трафика
    ├── services/               # Все микросервисы
    │   ├── proxy-service.yaml
    │   ├── events-service.yaml
    │   ├── movies-service.yaml
    │   ├── monolith.yaml
    │   └── postgres.yaml
    └── kafka/
        └── kafka.yaml          # Kafka + ZooKeeper
```

**Валидация и деплой:**

```bash
# Проверка синтаксиса
helm template cinemaabyss src/kubernetes/helm --dry-run

# Установка
helm install cinemaabyss src/kubernetes/helm --namespace cinemaabyss --create-namespace
```

**Результаты:**

- Helm chart валидируется без ошибок
- Все 7 подов развертываются и запускаются
- API доступен: `https://cinemaabyss.example.com/api/movies`
- Events API работает: `https://cinemaabyss.example.com/api/events/health`
- Postman тесты проходят успешно

**Скриншоты развертывания:**
![Статус подов](screenshots/helm%20up.png)
![API movies](screenshots/helm%20curl.png)
