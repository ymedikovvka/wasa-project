# WASAText

WASAText is a Web and Software Architecture project that implements a small WhatsApp/Telegram-style messaging application.

The project has two parts:

- A Go HTTP API that follows the OpenAPI specification in `doc/api.yaml`.
- A Vue.js frontend in `frontend/` that calls the Go API through Axios.

The login is intentionally simple because the course requires it only for testing and exam demonstration. It creates or selects a user by username and returns an identifier. It is not a secure authentication system.

## Project Structure

- `cmd/webapi/main.go`: application entry point. It opens the SQLite database, builds the API router, serves API routes, and serves the built frontend from `webui/`.
- `cmd/webapi/cors.go`: adds CORS headers so the API can be tested easily from browser tools and curl.
- `doc/api.yaml`: OpenAPI contract for all WASAText operations.
- `service/api/`: HTTP handlers, request validation, response mapping, and route registration.
- `service/database/`: in-memory and SQLite data layer used by the API.
- `service/database/schema.sql`: SQLite table definitions.
- `frontend/`: Vue 3 source code, Vite build config, Vue Router, Axios client, components, and styles.
- `webui/`: built Vue production files served by the Go server.
- `wasatext.db`: local SQLite database created by the application.

## Implemented API Operations

The API implements the operations required by `doc/api.yaml`:

- `doLogin`
- `setMyUserName`
- `setMyPhoto`
- `searchUsers`
- `getMyConversations`
- `startConversation`
- `getConversation`
- `sendMessage`
- `forwardMessage`
- `deleteMessage`
- `commentMessage`
- `uncommentMessage`
- `createGroup`
- `setGroupName`
- `setGroupPhoto`
- `addToGroup`
- `leaveGroup`

## How To Run

From the project root:

```bash
go run ./cmd/webapi
```

Then open:

```text
http://localhost:3000
```

If port `3000` is busy, run on another port:

```bash
env WASATEXT_ADDR=:3001 go run ./cmd/webapi
```

Then open:

```text
http://localhost:3001
```

The default database file is `wasatext.db`. To use a temporary database:

```bash
env WASATEXT_DB_PATH=/tmp/wasatext-demo.db WASATEXT_ADDR=:3001 go run ./cmd/webapi
```

## How To Develop The Vue Frontend

Install frontend dependencies:

```bash
cd frontend
npm install
```

Run the Vue dev server while the Go API is running on port `3000`:

```bash
npm run dev
```

Build the Vue app and copy it into the Go-served `webui/` directory:

```bash
npm run build
cp -R dist/. ../webui/
```

## How To Run With Docker

Build and run the app with Docker Compose:

```bash
docker compose up --build
```

Then open:

```text
http://localhost:3000
```

The Docker container stores SQLite data in a named volume called `wasatext-data`.

To run without Compose:

```bash
docker build -t wasatext .
docker run --rm -p 3000:3000 -v wasatext-data:/data wasatext
```

## How To Test

Run all Go tests:

```bash
go test ./...
```

Build the Vue frontend:

```bash
cd frontend
npm run build
```

Build the backend:

```bash
go build -o /tmp/wasa-webapi ./cmd/webapi
```

Run the API smoke test against a running server:

```bash
./scripts/api_smoke_test.sh http://localhost:3000
```

## Professor Demo Script

Use this flow during the exam:

1. Start the server and open the browser app.
2. Log in as `Medi`.
3. Log in as another user, for example `Lina`.
4. Switch back to `Medi` from Settings if needed.
5. Search for `Lina` in the People section.
6. Start a private chat with `Lina`.
7. Send a text message.
8. Send a photo message using the composer `+` button.
9. Switch to `Lina` and open the chat.
10. Reply to Medi's message.
11. Add a reaction to a message.
12. Clear your reaction.
13. Forward a message to another conversation.
14. Delete one of your own messages.
15. Create a group with the `+` button in the Groups section.
16. Rename the group.
17. Set a group photo.
18. Add another user to the group from the People section while the group is open.
19. Leave the group.

## What To Say About Login

This project does not implement real security login.

The `/session` endpoint receives a username. If the user exists, it returns the existing identifier. If not, it creates the user and returns the new identifier. The frontend stores this identifier and sends it in the `Authorization` header as `Bearer <identifier>`.

This follows the course requirement because it lets us create users and impersonate them for testing. It is useful for demonstrating conversations between users, but it is not password-based authentication and does not provide security.

## What To Say About Architecture

The API layer in `service/api/` receives HTTP requests, validates the request body and authorization identifier, calls the database layer, and converts internal Go structs into JSON responses.

The database layer in `service/database/` owns the application data model: users, conversations, members, messages, and reactions. The SQLite implementation persists data using the tables in `schema.sql`.

The frontend in `frontend/` is a Vue 3 single-page app built with Vite, Vue Router, and Axios. It does not contain fake data. It calls the same Go API that curl, tests, or the professor can inspect.

## Important Edge Cases

- Invalid usernames are rejected.
- Duplicate usernames are rejected when renaming.
- Protected routes require a valid user identifier.
- Users cannot manually add themselves to a new group; the creator is handled by the backend.
- Message delete applies to the user's own messages.
- Reactions are stored per message and per user.
- The group creation modal reloads the full user list before showing members, so it is not affected by previous search filters.
