# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Structure
- `backend/`: Go backend with Gin framework, MySQL database, JWT authentication
- `frontend/`: Next.js frontend with TypeScript, Tailwind CSS
- `db/`: Database initialization scripts

## Build/Test Commands
- Backend: `cd backend && go build ./cmd/api` and `go test ./internal/...`
- Frontend: `cd frontend && npm run build` and `npm run dev`
- Full stack: Use `docker-compose.yml` for development environment

## Testing
- Backend tests use `testutil.SetupTestDB()` for isolated database testing
- Frontend uses Jest for unit tests
- Run single backend test: `cd backend && go test -v ./internal/handlers -run TestFindAllUsers_AdminSuccess`

## Authentication & Authorization
- JWT tokens include user role ("user" or "admin")
- Admin routes require role check in handlers, not just middleware
- Frontend stores role in localStorage for UI conditional rendering

## API Patterns
- Backend responses validated with Zod schemas in frontend
- Error responses include "error" field for failed requests
- Admin endpoints under `/api/admin/` path

## Database
- MySQL with foreign key constraints
- User roles: "user" (default) or "admin"
- Passwords hashed with bcrypt, never returned in responses
