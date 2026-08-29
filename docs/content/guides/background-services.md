# Jobs, mail, and cache

Northframe supplies small adapter-friendly contracts for work that should not live in a page renderer. The in-memory implementations are useful for tests, local development, and a single process. Production systems that need durability or horizontal scaling can provide queue, mail, and cache adapters through the same interfaces.

## Create an in-memory job queue

```go
queue := jobs.NewMemory(jobs.Options{
    Workers:     4,
    Capacity:    256,
    MaxAttempts: 3,
    RetryDelay:  2 * time.Second,
    OnError: func(job jobs.Job, err error) {
        logger.Error("job failed", "name", job.Name, "error", err)
    },
})

queue.Handle("send-order-receipt", func(ctx context.Context, job jobs.Job) error {
    var payload orderReceiptJob
    if err := json.Unmarshal(job.Payload, &payload); err != nil {
        return err
    }
    return receipts.Send(ctx, payload.OrderID)
})

web.Provide[jobs.Queue](app, queue)
```

The queue is bounded and retries failed jobs. It is not durable: queued work is lost if the process exits. Use an external adapter implementing `jobs.Queue` when jobs must survive deployments or be shared across replicas.

## Enqueue after a request

```go
payload, err := json.Marshal(orderReceiptJob{OrderID: order.ID})
if err != nil {
    return web.ActionResult{}, err
}

queue := web.MustUse[jobs.Queue](ctx)
if err := queue.Enqueue(ctx.DetachedContext(), jobs.Job{
    Name:    "send-order-receipt",
    Payload: payload,
}); err != nil {
    return web.ActionResult{}, err
}
```

Enqueue a small identifier or event description, not the entire request context. The worker should reload authoritative data when it runs. If creating the order and publishing the job must be atomic, use an outbox table rather than hoping two separate operations both succeed.

## Schedule repeated work

```go
scheduler := jobs.NewScheduler(queue)
if err := scheduler.Every(15*time.Minute, jobs.Job{
    Name: "expire-unpaid-orders",
}); err != nil {
    log.Fatal(err)
}
```

The built-in scheduler runs in one process. In a multi-replica deployment, every replica would schedule the same job. Use leader election or an external scheduler when exactly-once scheduling matters, and make handlers idempotent even then.

## Send mail

Configure an SMTP sender:

```go
sender, err := mail.NewSMTP(mail.SMTPConfig{
    Address:  os.Getenv("SMTP_ADDRESS"),
    Host:     os.Getenv("SMTP_HOST"),
    Username: os.Getenv("SMTP_USERNAME"),
    Password: os.Getenv("SMTP_PASSWORD"),
    From:     "TopDuka <no-reply@example.com>",
})
if err != nil {
    log.Fatal(err)
}
web.Provide[mail.Sender](app, sender)
```

Use the interface in a service or job:

```go
sender := web.MustUse[mail.Sender](ctx)
err := sender.Send(ctx.StdContext(), mail.Message{
    To:      []string{account.Email},
    Subject: "Your order is confirmed",
    Text:    "Your order " + order.Number + " is confirmed.",
})
```

The mail package validates addresses, requires a subject and body, and rejects newline injection in headers. Prefer sending mail from a durable job so a slow provider does not hold the HTTP response open.

For tests, provide `*mail.Memory` and inspect `Outbox()`. This proves the recipient, subject, and body without connecting to SMTP.

## Cache typed values

```go
store := cache.NewMemory()
web.Provide[cache.Store](app, store)
```

Cache a serializable result:

```go
func featuredProducts(ctx context.Context, store cache.Store) ([]catalog.Product, error) {
    const key = "catalog:featured:v1"
    products, err := cache.GetJSON[[]catalog.Product](ctx, store, key)
    if err == nil {
        return products, nil
    }
    if !errors.Is(err, cache.ErrMiss) {
        return nil, err
    }

    products, err = catalog.LoadFeatured(ctx)
    if err != nil {
        return nil, err
    }
    if err := cache.SetJSON(ctx, store, key, products, 5*time.Minute); err != nil {
        return nil, err
    }
    return products, nil
}
```

Cache keys are application contracts. Include tenant IDs, permission scope, locale, and a schema version when those change the result. Delete or version a key after a mutation. Never cache one user's private data under a global key.

## Graceful shutdown

Stop accepting new requests, then give background services time to finish:

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

_ = scheduler.Shutdown(shutdownCtx)
_ = queue.Shutdown(shutdownCtx)
_ = server.Shutdown(shutdownCtx)
```

The exact order depends on the application. Usually stop schedules first, drain or stop workers, then close database and mail resources. Log jobs that could not finish so operators know what must be retried.
