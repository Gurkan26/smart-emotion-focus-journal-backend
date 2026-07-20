<div align="center">

<a href="https://academy.masterfabric.co">
  <img src="https://academy.masterfabric.co/academy-badge.png" width="120" alt="MasterFabric Academy">
</a>

<p>
  <sub>
    academy.masterfabric.co is a
    <a href="https://masterfabric.co">MasterFabric</a>
    subsidiary.
  </sub>
</p>

</div>

# Smart Emotion & Focus Journal - Backend API

This repository contains the Go-based REST API for the **Smart Emotion & Focus Journal** project. The backend is structured using Clean Architecture principles and configured as the official Go module: **`github.com/gurkanfikretgunak/masterfabric-go`**.

**GitHub Module Repository**: [https://github.com/gurkanfikretgunak/masterfabric-go](https://github.com/gurkanfikretgunak/masterfabric-go)  
**Live Backend API**: [https://smart-emotion-focus-journal-backend.onrender.com](https://smart-emotion-focus-journal-backend.onrender.com)  
**Vercel Frontend Live Demo**: [https://smart-emotion-focus-journal-fronten.vercel.app/auth](https://smart-emotion-focus-journal-fronten.vercel.app/auth)

## 🛠️ Technology Stack
- **Language**: Go (Golang)
- **Web Framework**: Gin Gonic
- **Database ORM**: GORM
- **Database Driver**: PostgreSQL Driver (pgx)
- **Security**: Bcrypt password hashing

---

## 📂 Project Structure (Clean Architecture)
The project layers separate concerns to ensure testing simplicity and dependency control:
- `config/`: Database initializers and environment configuration.
- `models/`: GORM struct models representing the database tables.
- `controllers/`: Handles business logic, input bindings, error logging, and database operations. Includes thread-safe in-memory maps/slices as fallbacks.
- `routes/`: Registers routes and maps them to their respective controllers.
- `main.go`: Application entry point setting up CORS middleware, auto-migrations, and serving the HTTP server.

---

## 💾 Database Schema
The GORM database contains 4 main tables:
1. **User**: Credentials data, mapping emails to hashed password values.
2. **UserConfig**: App settings including dashboard themes and notifications.
3. **Journal**: User reflections and parsed decision scores (`DecisionScore`) computed from local AI analysis.
4. **LlmMetric**: Performance tracking metrics mapping latency times, token volumes, and raw model output.

---

## 📡 Registered Routes (21 Endpoints)

The API supports 21 endpoints to handle authentication, telemetry logs, and configuration:

### 1. Root Route (1)
- `GET /` - Root status check (Operational API metadata metadata)

### 2. Auth Routes (8)
- `POST /register` - Register new user credentials (saves password hashed with Bcrypt)
- `POST /login` - Login check (returns a secure random session token)
- `POST /logout` - Terminates active session tokens
- `POST /refresh` - Refreshes token strings
- `GET /profile` - Retrieve profile information
- `PUT /profile` - Update profile information
- `PUT /password` - Modify active passwords
- `DELETE /delete` - Remove user profiles

### 3. Config Routes (2)
- `GET /config` - Retrieve user preferences (theme, notification status)
- `PUT /config` - Update config preferences

### 4. Web MLC-LLM / Inference Routes (7)
- `POST /api/journal` - Create a new journal entry and sync decision score
- `GET /api/journal` - Query journal logs list
- `POST /api/monitor/metrics` - Log run latency, tokens, and decision scores
- `GET /api/monitor/metrics` - Retrieve list of metrics logs for SVG charts
- `GET /api/monitor/scores` - Fetch cognitive load statistics summaries
- `POST /api/monitor/error` - Log inference runtime errors
- `DELETE /api/monitor/clear` - Wipe metric tables for the user

### 5. Common Routes (3)
- `GET /health` - Health check (returns 200 operational message)
- `GET /version` - App release version info
- `POST /feedback` - Submit user feedback logs

---

## 🚀 Installation and Run

### 1. Download Dependencies
```bash
go mod download
```

### 2. Start the Server Locally
```bash
go run main.go
```
The server will start by default on `http://localhost:8080`. 

> [!NOTE]
> **Database Fallback Mode**: If no local PostgreSQL database is running, the server logs a warning and automatically falls back to thread-safe **In-Memory Storage**. All auth registrations, logins, journals, and metrics will work perfectly in-memory during local frontend testing.
