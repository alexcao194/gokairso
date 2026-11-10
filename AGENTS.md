## DTO & Mapping
- Each feature keeps request, response, service, repository, and mapping structs in `dto.go`.
- All mapping functions and mapping logic must live in `dto.go`.

## Swagger
- Update Swagger annotations when changing handlers.
- Run:
  ```bash
  make swagger
  ```
- Every DTO `time.Time` field must include:
  ```go
  swaggertype:"string" format:"date-time"
  ```

## Repository
### Allowed
- Call `sqlc.Queries`
- Group simple read queries
- Handle DB errors (e.g. `pgx.ErrNoRows`)

### Forbidden
- Business logic
- Transaction management
- Direct SQL dependencies when sqlc exists

## Service
- Business logic belongs here.
- Transactions belong here.
- Repositories must never begin, commit, or rollback transactions.