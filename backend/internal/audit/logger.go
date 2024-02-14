package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Entry struct {
	ClusterID string
	Action    string
	Actor     string
	Resource  string
	Detail    interface{}
}

type LogEntry struct {
	ID        string          `json:"id"`
	ClusterID string          `json:"cluster_id"`
	Action    string          `json:"action"`
	Actor     string          `json:"actor"`
	Resource  string          `json:"resource"`
	Detail    json.RawMessage `json:"detail"`
	CreatedAt time.Time       `json:"created_at"`
}

type Logger struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func NewLogger(pool *pgxpool.Pool, log *zap.Logger) *Logger {
	return &Logger{pool: pool, log: log}
}

func (l *Logger) Record(ctx context.Context, e Entry) {
	actor := e.Actor
	if actor == "" {
		actor = "system"
	}

	var detail json.RawMessage
	if e.Detail != nil {
		b, _ := json.Marshal(e.Detail)
		detail = b
	}

	_, err := l.pool.Exec(ctx,
		`INSERT INTO audit_log (cluster_id, action, actor, resource, detail)
		 VALUES ($1, $2, $3, $4, $5)`,
		nilIfEmpty(e.ClusterID), e.Action, actor, e.Resource, detail,
	)
	if err != nil {
		// audit failures shouldn't blow up the caller
		l.log.Error("audit record failed", zap.Error(err))
	}
}

func (l *Logger) Query(ctx context.Context, clusterID string, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := l.pool.Query(ctx,
		`SELECT id, cluster_id, action, actor, resource, detail, created_at
		 FROM audit_log
		 WHERE ($1::uuid IS NULL OR cluster_id = $1)
		 ORDER BY created_at DESC
		 LIMIT $2`,
		nilIfEmpty(clusterID), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.ClusterID, &e.Action, &e.Actor, &e.Resource, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
