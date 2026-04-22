-- name: GetSetting :one
SELECT value FROM settings WHERE key = ?;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ListServices :many
SELECT id, title, url, icon_path, open_new_tab, position
FROM services
ORDER BY position ASC, id ASC;

-- name: GetService :one
SELECT id, title, url, icon_path, open_new_tab, position
FROM services
WHERE id = ?;

-- name: CreateService :one
INSERT INTO services (title, url, icon_path, open_new_tab, position)
VALUES (?, ?, ?, ?, ?)
RETURNING id, title, url, icon_path, open_new_tab, position;

-- name: UpdateService :exec
UPDATE services
SET title = ?, url = ?, open_new_tab = ?
WHERE id = ?;

-- name: DeleteService :exec
DELETE FROM services WHERE id = ?;

-- name: SetServicePosition :exec
UPDATE services SET position = ? WHERE id = ?;

-- name: SetServiceIconPath :exec
UPDATE services SET icon_path = ? WHERE id = ?;

-- name: MaxServicePosition :one
SELECT CAST(COALESCE(MAX(position), -1) AS INTEGER) AS max_position FROM services;

-- name: GetUserByUsername :one
SELECT id, username, password_hash FROM users WHERE username = ?;

-- name: CreateUser :one
INSERT INTO users (username, password_hash) VALUES (?, ?)
RETURNING id, username, password_hash;

-- name: CountUsers :one
SELECT COUNT(*) AS count FROM users;

-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetSession :one
SELECT s.token, s.user_id, s.expires_at, u.username
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = ?;

-- name: RefreshSession :exec
UPDATE sessions SET expires_at = ? WHERE token = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < ?;
