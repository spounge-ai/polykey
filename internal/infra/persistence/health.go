package persistence

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

// AlerterInterface defines a simple interface for sending alerts.
type AlerterInterface interface {
	SendAlert(level AlertLevel, message string, err error)
}

type ConnectionMonitor struct {
	pool      *pgxpool.Pool
	alerter   AlerterInterface
	mu        sync.RWMutex
	isHealthy bool
}

func NewConnectionMonitor(pool *pgxpool.Pool, alerter AlerterInterface) *ConnectionMonitor {
	return &ConnectionMonitor{
		pool:      pool,
		alerter:   alerter,
		isHealthy: true, // Assume healthy on startup
	}
}

func (cm *ConnectionMonitor) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.performHealthCheck(ctx)
		}
	}
}

func (cm *ConnectionMonitor) performHealthCheck(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := cm.pool.Ping(checkCtx)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err != nil {
		if cm.isHealthy {
			cm.isHealthy = false
			if cm.alerter != nil {
				cm.alerter.SendAlert(AlertLevelCritical, "Database connection unhealthy", err)
			}
		}
	} else {
		if !cm.isHealthy {
			cm.isHealthy = true
			if cm.alerter != nil {
				cm.alerter.SendAlert(AlertLevelInfo, "Database connection recovered", nil)
			}
		}
	}
}

func (cm *ConnectionMonitor) IsHealthy() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isHealthy
}
