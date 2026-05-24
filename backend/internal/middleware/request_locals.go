package middleware

const (
	// LocalUserIDKey stores the authenticated user identifier in Fiber locals.
	LocalUserIDKey = "user_id"
	// LocalSessionKey stores a hashed request session token in Fiber locals.
	LocalSessionKey = "session_key"
)
