# golang-url-shortener
A production-style URL shortening REST API built using Go, PostgreSQL, Redis, and Docker.

The service generates short URLs, performs fast redirect resolution using a cache-aside strategy, and is designed to simulate real backend system architecture.
# Features:-
-URL creation and validation
-Short hash generation
-HTTP redirect resolution
-Cache-aside pattern using Redis
-PostgreSQL persistent storage
-Health check endpoint
-Fully Dockerized deployment
-High-throughput performance tested

# Single command deployment:
-  docker-compose up --build

This launches:
-  Go API Server
-  PostgreSQL database
-  Redis cache
# Architecture
Client Request
      │
      ▼
Go API Server
      │
      ├── Redis (Cache Hit → Instant Redirect)
      │
      └── PostgreSQL (Cache Miss → DB Lookup → Cache Populate)
