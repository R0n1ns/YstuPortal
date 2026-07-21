package api

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Metrics struct {
	startTime        time.Time
	totalRequests    atomic.Uint64
	totalErrors      atomic.Uint64
	inFlight         atomic.Int64
	totalDurationNs  atomic.Uint64
	statusCountsLock sync.Mutex
	statusCounts     map[int]uint64
}

func NewMetrics() *Metrics {
	return &Metrics{
		startTime:    time.Now(),
		statusCounts: make(map[int]uint64),
	}
}

func (m *Metrics) Middleware() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		m.inFlight.Add(1)
		start := time.Now()
		err := ctx.Next()
		duration := time.Since(start)
		m.inFlight.Add(-1)

		status := ctx.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			}
		}
		m.totalRequests.Add(1)
		m.totalDurationNs.Add(uint64(duration.Nanoseconds()))
		if status >= 400 {
			m.totalErrors.Add(1)
		}

		m.statusCountsLock.Lock()
		m.statusCounts[status]++
		m.statusCountsLock.Unlock()

		return err
	}
}

func (m *Metrics) Handler() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		uptime := time.Since(m.startTime)
		total := m.totalRequests.Load()
		avgMs := float64(0)
		if total > 0 {
			avgMs = float64(m.totalDurationNs.Load()) / float64(total) / float64(time.Millisecond)
		}

		m.statusCountsLock.Lock()
		statusSnapshot := make(map[int]uint64, len(m.statusCounts))
		for code, count := range m.statusCounts {
			statusSnapshot[code] = count
		}
		m.statusCountsLock.Unlock()

		return ctx.JSON(fiber.Map{
			"uptime_ms":          uptime.Milliseconds(),
			"total_requests":     total,
			"total_errors":       m.totalErrors.Load(),
			"in_flight":          m.inFlight.Load(),
			"avg_duration_ms":    avgMs,
			"status_code_counts": statusSnapshot,
		})
	}
}
