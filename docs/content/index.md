# Northframe

Northframe is a Go-first full-stack framework for server-rendered applications. Inspired by Astro's server-first composition and Svelte's approachable component authoring, a `.north` file describes HTML, typed Go props, and optional browser TypeScript. Northframe compiles it into Go renderers, route registration, CSS, and small browser modules, producing a normal Go program that can be deployed as one executable.

Northframe is designed for teams that want the ergonomics of a batteries-included web framework without making Node, Deno, Svelte, or a JavaScript server part of production. Go owns requests, data access, authentication, jobs, and HTML rendering. TypeScript is available where the browser needs interaction.

The current release is **v0.1.0-beta**. The beta is intended for evaluation, prototypes, and real migration testing. It is not a promise of `v1` API stability; pin the exact version, read compiler diagnostics carefully, and review the compatibility notes before production use.

## Start here

- [Install Northframe](/getting-started/installation)
- [Build your first application](/getting-started/first-app)
- [Understand the project structure](/getting-started/project-structure)
- [Write a route and layout](/concepts/routes-and-layouts)
- [Pass data with `Props`](/concepts/props)
- [Handle forms and actions](/guides/forms-and-actions)

The calculator example is the smallest complete example. The commerce example demonstrates PostgreSQL, sqlc, typed forms, API routes, browser dependencies, and a larger route tree.

## The core idea

```text
.north + .north.go + CSS + optional TypeScript
                    │
                    ▼
          northframe generate / northframe build
                    │
                    ▼
        generated Go renderer + route tree
                    │
                    ▼
                 one Go binary
```

## Follow one request

A rendered page crosses four explicit boundaries:

- The filesystem route chooses the URL and layout chain.
- A Go loader receives `*web.Context` and calls application services.
- The loader returns a generated, compile-time checked `PageProps` value.
- The generated renderer evaluates Go control flow, escapes values, and writes HTML.

Browser TypeScript is a fifth, optional boundary. It starts after SSR and owns only client interaction. It does not load server props directly or replace Go actions. When browser state must be persisted, submit a form action or call an API route.

This separation is the main Northframe mental model. Templates do not query databases. Components do not secretly fetch during rendering. Go services do not know how a modal is animated. Generated code connects these pieces and is safe to delete and recreate.

## Choose a learning path

For a first application, read Installation, Your first app, Routes and layouts, Props and rendering, Components, and Forms and actions in that order. That path explains enough to build a server-rendered CRUD feature with typed data and progressive loading states.

For a production application, continue through Request context, Database and sqlc, Authentication, Middleware and security, Testing, and Deployment. Add API routes when another client needs JSON, and Jobs, mail, and cache when work should move outside the response.

Framework contributors and editor authors should use the reference section. The syntax page defines accepted template forms, the CLI page documents command ownership, and the editor page explains which intelligence comes from Northframe, gopls, embedded TypeScript, HTML, and Tailwind.

## What Northframe does not hide

Northframe does not invent a new database model, service container language, authentication database, or deployment platform. It provides typed Go interfaces and sensible built-in adapters while keeping application choices visible:

- sqlc and `database/sql` remain normal Go database code.
- `net/http` middleware can run directly.
- sessions can use an application-owned database or cache.
- browser dependencies are compiled without becoming server dependencies.
- the final program remains a normal Go main package.

This means more decisions remain in application code than in a magic configuration file, but those decisions are testable with ordinary Go tools and do not disappear behind generated runtime behavior.

## How to read this guide

If you are new to Northframe, read the getting-started pages in order. Then use the concepts pages to understand the template language and the guides when adding production concerns. The reference pages are searchable descriptions of commands and supported syntax.

Every tutorial follows the same loop: edit a source file, run `northframe generate` or `northframe run`, inspect the browser result, and read any compiler diagnostic as the next correction. Northframe is in beta and still moving quickly, so the compiler, the documentation for your pinned tag, and `northframe help` remain authoritative if behavior changes.
