# Production operations

Northframe keeps production services explicit and adapter-friendly. Applications opt in to the pieces they need; no global container or background process starts implicitly.

## Request logs and traces

```go
logger := observability.JSONLogger(slog.LevelInfo)
app.Use(observability.HTTP(logger))
```

The middleware emits one structured `http.request` record with method, path, status, response bytes, duration, request ID, and trace ID. It accepts a valid W3C `traceparent`, creates one when absent, and returns both `traceparent` and `X-Request-ID`. Retrieve correlation values with `observability.TraceID(ctx)` and `observability.RequestID(ctx)`.

## Cache

`cache.Store` is the application boundary for Redis, Memcached, or another shared cache. `cache.NewMemory()` is concurrency-safe and supports TTLs for development and single-instance deployments. `cache.SetJSON` and `cache.GetJSON` preserve typed application values.

## Mail

Depend on `mail.Sender`. `mail.NewSMTP` is the standard SMTP adapter and `mail.Memory` is a test outbox. Messages validate senders, recipients, subjects, and bodies before delivery.

## Queues and schedules

`jobs.Queue` is the boundary for durable adapters. `jobs.NewMemory` runs a bounded worker pool with delayed retries and terminal error reporting. Register handlers by name, enqueue serializable payloads, and shut the worker down with the application context. `jobs.NewScheduler(queue).Every(...)` sends recurring work through the same queue so schedules never bypass retry and monitoring policy.

## Deployment

Validate the exact production build:

```sh
north deploy check
```

Generate a non-root, multi-stage container definition:

```sh
north deploy docker -output Dockerfile
```

The generated container exposes `PORT=8000`, includes an `/api/health` probe, and contains only the compiled application and Alpine runtime. Secrets remain runtime environment variables and are never copied into the image by Northframe.
