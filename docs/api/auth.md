# Auth

All protected endpoints require a `Authorization: Bearer <token>` header. Tokens are obtained via login.

---

## `POST /api/auth/login`

No auth required.

**Request:**

```json
{
  "username": "admin",
  "password": "password"
}
```

**Response (200):**

```json
{
  "token": "eyJ..."
}
```

---

## `GET /api/auth/me`

Get the currently authenticated user.

**Response (200):**

```json
{
  "username": "admin"
}
```

---

## `POST /api/auth/logout`

Revoke the current token.

**Response (200):** No content.

---

## `POST /api/auth/credentials`

Change username or password.

**Request:**

```json
{
  "currentPassword": "old",
  "newUsername": "admin",
  "newPassword": "new"
}
```

**Response (200):** No content.

---
