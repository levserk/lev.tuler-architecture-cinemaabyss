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

**1. Proxy Service (Strangler Fig):**

- API Gateway на порту 8000
- Постепенная миграция трафика через `MOVIES_MIGRATION_PERCENT`
- Все тесты proxy работают

**2. Events Service (Kafka MVP):**

- Producer/Consumer для топиков movies, users, payments
- API endpoints: `/api/events/health`, `/api/events/movie`, `/api/events/user`, `/api/events/payment`

**Результат:** все Postman тесты зеленые (22/22) ✅

# Ответ на Задание 3

**CI/CD:** Доработал `docker-build-push.yml` для сборки proxy и events сервисов в GitHub Container Registry

**Kubernetes:** Создал манифесты `events-service.yaml` и `proxy-service.yaml`, настроил ingress для доступа к событиям

**Деплой по шагам 1-12:**

- Все 7 подов запущены ✅
- API работает: `https://cinemaabyss.example.com/api/movies` ✅
- Newman тесты: 22/22 успешны ✅

**Скриншоты:**
![API вызов](screenshots/curl.png)
![Events логи](screenshots/events-service%20logs.png)

# Ответ на Задание 4

**Helm charts:** Обновил `values.yaml` со своими образами, заполнил шаблоны `proxy-service.yaml` и `events-service.yaml`

**Деплой:** `helm install cinemaabyss src/kubernetes/helm --namespace cinemaabyss --create-namespace`

**Результат:**

- Все 7 подов запущены ✅
- `https://cinemaabyss.example.com/api/movies` работает ✅
- Postman тесты проходят ✅

**Скриншоты развертывания:**
![Статус подов](screenshots/helm%20up.png)
![API movies](screenshots/helm%20curl.png)
