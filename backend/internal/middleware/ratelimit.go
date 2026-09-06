package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type loginAttempt struct {
	Count       int
	LastTry     time.Time
	LockedUntil time.Time
}

var (
	loginAttempts = make(map[string]*loginAttempt)
	loginMutex    sync.RWMutex
)

const (
	MaxLoginAttempts    = 5
	InitialLockDuration = 1 * time.Minute
	MaxLockDuration     = 15 * time.Minute
	AttemptWindow       = 5 * time.Minute

	// maxBackoffExponent bounds the exponential-backoff shift. InitialLockDuration
	// << 10 already exceeds MaxLockDuration, so clamping here changes no observable
	// behaviour while preventing the shift from overflowing.
	maxBackoffExponent = 10
)

// AuthRateLimiter provides brute-force protection for authentication endpoints
func AuthRateLimiter(c *fiber.Ctx) error {
	ip := c.IP()

	loginMutex.Lock()

	attempt, exists := loginAttempts[ip]
	now := time.Now()

	if !exists {
		loginAttempts[ip] = &loginAttempt{
			Count:   0,
			LastTry: now,
		}
		loginMutex.Unlock()
		return c.Next()
	}

	// Check if currently locked
	if now.Before(attempt.LockedUntil) {
		loginMutex.Unlock()
		remaining := attempt.LockedUntil.Sub(now).Seconds()
		return c.Status(429).JSON(fiber.Map{
			"error":            "Too many failed login attempts. Please try again later.",
			"code":             "rate_limited",
			"retry_after_secs": int(remaining),
		})
	}

	// Reset counter if outside the attempt window
	if now.Sub(attempt.LastTry) > AttemptWindow {
		attempt.Count = 0
	}

	loginMutex.Unlock()
	return c.Next()
}

// RecordFailedLogin should be called after a failed login attempt
func RecordFailedLogin(ip string) {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	attempt, exists := loginAttempts[ip]
	now := time.Now()

	if !exists {
		loginAttempts[ip] = &loginAttempt{
			Count:   1,
			LastTry: now,
		}
		return
	}

	attempt.Count++
	attempt.LastTry = now

	// Cap the counter so a persistent prober cannot run it up unbounded.
	if attempt.Count > MaxLoginAttempts+maxBackoffExponent {
		attempt.Count = MaxLoginAttempts + maxBackoffExponent
	}

	if attempt.Count >= MaxLoginAttempts {
		// Exponential backoff, computed with a clamped exponent so the shift
		// can never overflow time.Duration and wrap to a zero/negative lock.
		exponent := attempt.Count - MaxLoginAttempts
		if exponent > maxBackoffExponent {
			exponent = maxBackoffExponent
		}
		lockDuration := InitialLockDuration << uint(exponent)
		if lockDuration <= 0 || lockDuration > MaxLockDuration {
			lockDuration = MaxLockDuration
		}
		attempt.LockedUntil = now.Add(lockDuration)
	}
}

// RecordSuccessfulLogin should be called after a successful login
func RecordSuccessfulLogin(ip string) {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	delete(loginAttempts, ip)
}

// CleanupOldAttempts removes stale entries from the map (call periodically)
func CleanupOldAttempts() {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	now := time.Now()
	for ip, attempt := range loginAttempts {
		// Remove entries that are both unlocked and haven't been used recently
		if now.After(attempt.LockedUntil) && now.Sub(attempt.LastTry) > 30*time.Minute {
			delete(loginAttempts, ip)
		}
	}
}

func init() {
	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			CleanupOldAttempts()
		}
	}()
}
