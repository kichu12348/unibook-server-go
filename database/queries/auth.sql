-- name: GetUserByEmail :one
-- Check if a user with a given email already exists
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetCollegeByID :one
-- Get college details to validate the email domain
SELECT * FROM colleges
WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
-- Insert a new user into the database
INSERT INTO users (
  full_name, email, password_hash, role, college_id, approval_status
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CreateForumHead :one
-- Create an entry in the forum_heads join table
INSERT INTO forum_heads (
  user_id, forum_id, is_verified
) VALUES (
  $1, $2, false
)
RETURNING *;

-- name: SetUserEmailVerificationDetails :exec
-- Sets the OTP token and expiration for a user after registration
UPDATE users
SET 
  email_verification_token = $2,
  email_verification_expires = $3
WHERE id = $1;

-- name: VerifyUserEmail :one
-- Marks a user's email as verified and clears the token fields
UPDATE users
SET
  is_email_verified = true,
  email_verification_token = NULL,
  email_verification_expires = NULL
WHERE id = $1
RETURNING *;

-- name: GetSuperAdminByEmail :one
-- Fetches a super admin for login verification
SELECT * FROM super_admins
WHERE email = $1 LIMIT 1;

-- name: SetUserPasswordResetDetails :exec
-- Sets the user password reset token
UPDATE users
SET
  password_reset_token=$2,
  password_reset_expires=$3
WHERE id=$1;


-- name: UpdateUserPassword :exec
-- Sets the new hashed password for user after reseting
UPDATE users
SET
  password_hash=$2,
  password_reset_expires=NULL,
  password_reset_token=NULL
WHERE id=$1;

-- name: GetUserByID :one
SELECT
  u.id,
  u.full_name AS "fullName",
  u.email,
  u.role,
  u.college_id AS "collegeId",
  u.approval_status AS "approvalStatus",
  u.is_email_verified AS "isEmailVerified",
  u.created_at AS "createdAt",

  json_build_object(
    'id', c.id,
    'name', c.name
  ) AS college,

  COALESCE(
    json_agg(
      json_build_object(
        'forumId', fh.forum_id,
        'isVerified', fh.is_verified,
        'forum', json_build_object('name', f.name)
      )
    ) FILTER (WHERE fh.user_id IS NOT NULL),
    '[]'::json
  ) AS forum_heads
FROM
  "users" AS u
LEFT JOIN "colleges" AS c ON u.college_id = c.id
LEFT JOIN "forum_heads" AS fh ON u.id = fh.user_id
LEFT JOIN "forums" AS f ON fh.forum_id = f.id
WHERE
  u.id = $1
GROUP BY
  u.id, c.id
LIMIT 1;


-- name: GetSuperAdminByID :one
SELECT * FROM super_admins WHERE id = $1 LIMIT 1;