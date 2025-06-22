# Limestone Chat Service

---

Welcome to the Limestone Chat Service! This is the real-time messaging backend for the Limestone project, providing robust, conversation-based chat functionality via WebSockets. It's designed to integrate seamlessly with your existing Limestone user authentication.

---

## Features

* **Real-time Communication:** Built on WebSockets for instant message delivery.
* **Conversation-Centric Design:** Messages are organized into distinct **Conversation ID**s, enabling clear chat histories for various contexts.
* **Purpose-Driven Chats:** Supports different chat purposes like **`nikkah_service`** (for marriage-related queries), **`revert_service`** (for issue resolution), and **`general_chat`** (for general communication).
* **1-on-1 Private Chats:** Ensures secure and private conversations between two specific users.
* **Message Persistence:** All chat data, including messages and conversation details, is reliably stored in a PostgreSQL database.
* **Efficient Hub Architecture:** A central `Hub` manages all active WebSocket connections, efficiently broadcasting messages to the correct recipients.
* **Robust & Resilient:** Includes comprehensive error handling for WebSocket operations, message processing, and database interactions, plus a ping/pong heartbeat to maintain connection stability.

---

## Technologies Used

* **Go:** The core programming language.
* **Gorilla WebSocket:** For WebSocket implementation in Go.
* **GORM:** An Object-Relational Mapper for database operations.
* **PostgreSQL:** The primary database for storing chat data.
* **UUID:** For generating unique identifiers for all chat entities.

---

## Getting Started

Follow these steps to get the Limestone Chat Service up and running locally.

### Prerequisites

* Go (version 1.18+ recommended)
* PostgreSQL
* Git
* A tool for WebSocket testing (e.g., Postman, WebSocket King, Insomnia)
* Access to the **Limestone Auth Service** to obtain JWT tokens.

### 1. Clone the Repository

```bash
git clone [https://github.com/masjids-io/limestone-chat.git](https://github.com/masjids-io/limestone-chat.git)
cd limestone-chat
```

### 2. Set Up Environment Variables
Create a .env file in the root of your project or set these variables directly in your shell environment.

```bash
export ACCESS_SECRET="your_jwt_access_secret_from_limestone"
export REFRESH_SECRET="your_jwt_refresh_secret_from_limestone"
export ACCESS_EXPIRATION="5m" # e.g., 5 minutes for access tokens
export REFRESH_EXPIRATION="168h" # e.g., 1 week for refresh tokens
export DATABASE_URL="host=localhost user=postgres password=your_db_password dbname=limestone port=5432 sslmode=disable"
```
### 3. Database Setup
Ensure your PostgreSQL instance is running. The service expects a database named limestone. If it doesn't exist, create it. The chat service will automatically run database migrations on startup, so you don't need to apply schemas manually.

### 4. Get your JWT token
Before connecting to the chat service, you must log in to your Limestone main service to obtain a valid JWT ACCESS_TOKEN. This token will be used to authenticate your WebSocket connection.

### 5. Run the service
Navigate to the project root directory in your terminal and execute:
```bash
go run cmd/main.go
```
You should see log messages confirming successful database connection and the server starting (by default on port 8082).

## Endpoints
### WebSocket Connection
Connect to this endpoint to establish a real-time chat session.

* Endpoint: ws://localhost:8082/ws
* Query Parameters:
  * purpose: (Required) Specifies the context of the conversation. Examples: nikkah_service, revert_service, general_chat.
  * partner_id: (Required for 1-on-1 chats) The UUID of the specific user you want to chat with.
  * Headers:
  Authorization: Bearer <YOUR_JWT_ACCESS_TOKEN> (The token obtained from Limestone login)

### Example Connection URLs:
* User A (UUID: 29838a14-b888-42ad-825c-1ef65e3599a8) wants to chat with User B (UUID: bf6f7fff-577e-4e1d-9d03-ead0a9ec69ad) about nikkah_service:
```bash
ws://localhost:8082/ws?purpose=nikkah_service&partner_id=bf6f7fff-577e-4e1d-9d03-ead0a9ec69ad
```

## Message Formats

#### 1. Sample Message (Client to Server)
This is the JSON payload you send over the WebSocket connection.
* For nikkah_service :
```json
{
  "type": "text",
  "content": "Assalamu'alaikum. I'd like to ask about the status of my marriage application.",
  "media_url": "",
  "metadata": {},
  "reply_to_message_id": null
}
```
* For revert_service
```json
{
  "type": "text",
  "content": "I'd like to request a revert for transaction number INV12345. Please help.",
  "media_url": null,           
  "metadata": {
    "transaction_id": "INV12345",
    "reason": "Incorrect amount entered"
  },            
  "reply_to_message_id": null
}
```
* For general_chat
```json
{
  "type": "text",
  "content": "Okay, I've forwarded it to the relevant team. Please await further updates.",
  "media_url": "",           
  "metadata": {},            
  "reply_to_message_id": "521c1212-a788-4c99-b6e2-84c5f8d266fe" 
}
```

#### 2. Sample Response (Server to Client)
This is the JSON payload you will receive from the WebSocket connection after a message is sent and processed.
* Generic Response:
```json
{
  "id": "c9af6dcf-0797-4d27-aa44-00d55f4b5630",
  "conversation_id": "6adbcc4d-5534-4347-8f13-166580f02eec",
  "sender_id": "29838a14-b888-42ad-825c-1ef65e3599a8",
  "content": "Assalamu'alaikum. I'd like to ask about the status of my marriage application.",
  "type": "text",
  "media_url": null,
  "metadata": {},
  "reply_to_message_id": null,
  "created_at": "2025-06-22T11:18:49+08:00"
}
```