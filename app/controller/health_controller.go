package controller

import (
	"chess-engine/app/pkg"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// checkTimeout bounds each dependency probe. A health check that blocks on a
// wedged backend is worse than one that reports the backend as down.
const checkTimeout = 2 * time.Second

type HealthController interface {
	Check(c *gin.Context)
}

type HealthControllerImpl struct {
	db          *gorm.DB
	redisClient *pkg.RedisClient
}

func HealthControllerInit(db *gorm.DB, redisClient *pkg.RedisClient) *HealthControllerImpl {
	return &HealthControllerImpl{db: db, redisClient: redisClient}
}

// Check reports liveness of the process plus its two hard dependencies, and
// answers 503 if either is unreachable. It deliberately does not use respond():
// the status code is the payload as far as Docker and uptime probes care.
func (h HealthControllerImpl) Check(c *gin.Context) {
	checks := gin.H{
		"postgres": statusOf(h.pingDB(c.Request.Context())),
		"redis":    statusOf(h.pingRedis(c.Request.Context())),
	}

	code := http.StatusOK
	status := "ok"
	for _, v := range checks {
		if v != "ok" {
			code = http.StatusServiceUnavailable
			status = "degraded"
			break
		}
	}

	c.JSON(code, gin.H{"status": status, "checks": checks})
}

func (h HealthControllerImpl) pingDB(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

func (h HealthControllerImpl) pingRedis(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	return h.redisClient.Ping(ctx)
}

func statusOf(err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return "ok"
}
