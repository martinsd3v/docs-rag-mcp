# ADR-001: Database Selection for User Service

**Status:** Accepted
**Version:** v1.0
**Created:** 2024-02-01
**Author:** Database Team

## Context

We need to select a database for the User Service. This service handles user authentication, profile management, and session storage.

## Decision Drivers

- **Performance:** Low latency for authentication queries
- **Scalability:** Handle millions of users
- **Consistency:** Strong consistency for user data
- **Operations:** Easy to operate and backup

## Options Considered

### Option 1: PostgreSQL

Mature relational database with excellent ACID compliance and JSON support.

**Pros:**
- Strong consistency
- Rich query capabilities
- Mature ecosystem
- Easy horizontal scaling with read replicas

**Cons:**
- Write scaling requires sharding
- More complex operational model than NoSQL

### Option 2: MongoDB

Document database with flexible schema and built-in sharding.

**Pros:**
- Flexible schema
- Built-in horizontal scaling
- Good for hierarchical data

**Cons:**
- Eventual consistency by default
- Less mature transactional support

### Option 3: CockroachDB

Distributed SQL database with automatic sharding.

**Pros:**
- SQL interface with automatic scaling
- Strong consistency
- No manual sharding

**Cons:**
- Higher latency than single-node databases
- Newer technology with less ecosystem

## Decision

We will use **PostgreSQL** as the primary database for the User Service.

### Rationale

1. Strong consistency is critical for user data
2. Team has extensive PostgreSQL experience
3. Excellent tooling for backups and monitoring
4. Can scale reads with replicas initially

## Implementation

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

## Consequences

- Need to implement connection pooling (PgBouncer)
- Must plan for sharding if write load exceeds single-node capacity
- Training required for team members unfamiliar with PostgreSQL
