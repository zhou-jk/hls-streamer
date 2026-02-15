package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

type visitor struct {
	count    int
	resetAt  time.Time
}

func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(window)
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.After(v.resetAt) {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists || time.Now().After(v.resetAt) {
			rl.visitors[ip] = &visitor{
				count:   1,
				resetAt: time.Now().Add(window),
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		v.count++
		if v.count > rl.rate {
			rl.mu.Unlock()
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()

		c.Next()
	}
}
