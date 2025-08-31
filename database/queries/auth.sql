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
    "users"."id" AS "id",
    "users"."full_name" AS "fullName",
    "users"."email" AS "email",
    "users"."role" AS "role",
    "users"."college_id" AS "collegeId",
    "users"."approval_status" AS "approvalStatus",
    "users"."is_email_verified" AS "isEmailVerified",
    "users"."created_at" AS "createdAt",
    -- This is the only line that has changed
    COALESCE("users_college"."data", 'null'::json) AS "college",
    "users_forum_heads"."data" AS "forumHeads"
FROM "users" "users"
LEFT JOIN LATERAL (
    SELECT 
        json_build_object(
            'id', "users_college"."id",
            'name', "users_college"."name"
        )::json AS "data"
    FROM (
        SELECT * FROM "colleges" "users_college"
        WHERE "users_college"."id" = "users"."college_id"
        LIMIT 1
    ) "users_college"
) "users_college" ON TRUE
LEFT JOIN LATERAL (
    SELECT 
        COALESCE(
            json_agg(
                json_build_object(
                    'forumId', "users_forum_heads"."forum_id",
                    'isVerified', "users_forum_heads"."is_verified",
                    'forum', "users_forum_heads_forum"."data"
                )
            ),
            '[]'::json
        ) AS "data"
    FROM "forum_heads" "users_forum_heads"
    
    LEFT JOIN LATERAL (
        SELECT 
            json_build_object(
                'name', "users_forum_heads_forum"."name"
            )::json AS "data"
        FROM (
            SELECT * FROM "forums" "users_forum_heads_forum"
            WHERE "users_forum_heads_forum"."id" = "users_forum_heads"."forum_id"
            LIMIT 1
        ) "users_forum_heads_forum"
    ) "users_forum_heads_forum" ON TRUE
    
    WHERE "users_forum_heads"."user_id" = "users"."id"
) "users_forum_heads" ON TRUE
WHERE "users"."id" = $1
LIMIT 1;


-- name: GetSuperAdminByID :one
SELECT * FROM super_admins WHERE id = $1 LIMIT 1;