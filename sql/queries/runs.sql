-- name: CreateRun :one
INSERT INTO runs (name, title, description, docker_image, parameters, data, mounts, created_at, user_id)
VALUES (?,?,?,?,?,?,?,datetime('now'),?)
RETURNING *;

-- name: GetRun :one
SELECT r.* FROM runs r
WHERE r.id = ? AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR r.user_id = ?
);

-- name: DeleteRun :exec
DELETE FROM runs
WHERE runs.id = ? AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR runs.user_id = ?
);

-- name: StartRun :one
UPDATE runs
SET status = 'running', started_at = datetime('now')
WHERE runs.id = ? AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR runs.user_id = ?
)
RETURNING *;

-- name: FinishRun :one
UPDATE runs SET status = 'finished', finished_at = datetime('now')
WHERE runs.id = ?
RETURNING *;

-- name: RunErrored :one
UPDATE runs SET status = 'errored', error_message = ?, finished_at = datetime('now'), has_errored = TRUE
WHERE runs.id = ?
RETURNING *;

-- name: SetRunGotapMetadata :one
UPDATE runs SET gotap_metadata = ?
WHERE runs.id = ?
RETURNING *;

-- name: GetAllRuns :many
SELECT r.* FROM runs r
WHERE (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
   OR r.user_id = ?;

-- name: GetIdleRuns :many
SELECT r.* FROM runs r
WHERE r.status = 'pending' AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR r.user_id = ?
);

-- name: GetRunning :many
SELECT r.* FROM runs r
WHERE r.status = 'running' AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR r.user_id = ?
);

-- name: GetFinishedRuns :many
SELECT r.* FROM runs r
WHERE r.status = 'finished' AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR r.user_id = ?
);

-- name: GetErroredRuns :many
SELECT r.* FROM runs r
WHERE r.status = 'errored' AND (
  (SELECT u.is_admin FROM users u WHERE u.id = ?) = TRUE 
  OR r.user_id = ?
);
