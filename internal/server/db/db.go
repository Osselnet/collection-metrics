package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"time"
)

const (
	initTimeOut  = 2 * time.Second
	queryTimeOut = 1 * time.Second

	queryInsertGauge   = `INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)`
	queryInsertCounter = `INSERT INTO metrics (id, type, delta) VALUES ($1, 'counter', $2)`
	queryUpdateGauge   = `UPDATE metrics SET value = $2 WHERE id = $1`
	queryUpdateCounter = `UPDATE metrics SET delta = $2 WHERE id = $1`
	queryGet           = `SELECT id, type, value, delta FROM metrics WHERE id=$1`
	queryGetMetrics    = `SELECT id, type, value, delta FROM metrics`
)

type DateBaseStorage interface {
	Put(context.Context, string, interface{}) error
	Get(context.Context, string) (interface{}, error)
	PutMetrics(context.Context, metrics.Metrics) error
	GetMetrics(context.Context) (metrics.Metrics, error)

	Ping(parentCtx context.Context) error
	Shutdown() error
}

type MemStorageDb struct {
	db *sql.DB
}

type metricsDB struct {
	ID    string
	MType string
	Value sql.NullFloat64
	Delta sql.NullInt64
}

type Sender func(context.Context) error
type QueryContext func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
type QueryRowContext func(ctx context.Context, query string, args ...any) *sql.Row
type ExecContext func(ctx context.Context, query string, args ...any) (sql.Result, error)

func (s *MemStorageDb) Put(parentCtx context.Context, id string, val interface{}) error {
	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	switch m := val.(type) {
	case metrics.Gauge:
		err := s.putGauge(ctx, id, m)
		if err != nil {
			return err
		}
	case metrics.Counter:
		err := s.putCounter(ctx, id, m)
		if err != nil {
			return err
		}
	default:
		return errors.New("storage: metric not implemented")
	}
	return nil
}

func (s *MemStorageDb) putGauge(ctx context.Context, id string, val metrics.Gauge) error {
	fn := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
	result, err := fn(ctx, queryUpdateGauge, id, val)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		fn := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
		_, err = fn(ctx, queryInsertGauge, id, val)

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *MemStorageDb) putCounter(ctx context.Context, id string, val metrics.Counter) error {
	fn := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
	result, err := fn(ctx, queryGet, id)

	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		fn1 := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
		_, err = fn1(ctx, queryInsertCounter, id, val)

		if err != nil {
			return err
		}
		return nil
	}

	mdb := metricsDB{}
	fn3 := RetryQueryRowContext(s.db.QueryRowContext, 3, 1*time.Second)
	row := fn3(ctx, queryGet, id)
	err = row.Scan(&mdb.ID, &mdb.MType, &mdb.Value, &mdb.Delta)

	if err != nil {
		return err
	}

	if !mdb.Delta.Valid {
		mdb.Delta.Int64 = 0
	}

	count := val + metrics.Counter(mdb.Delta.Int64)

	fn2 := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
	_, err = fn2(ctx, queryUpdateCounter, id, count)

	if err != nil {
		return err
	}
	return nil
}

func (s *MemStorageDb) Get(parentCtx context.Context, id string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	m, err := s.getId(ctx, id)
	if err != nil {
		return nil, err
	}

	switch m.MType {
	case metrics.TypeGauge:
		if !m.Value.Valid {
			return nil, fmt.Errorf("NULL gauge value")
		}
		return metrics.Gauge(m.Value.Float64), nil
	case metrics.TypeCounter:
		if !m.Delta.Valid {
			return nil, fmt.Errorf("NULL counter value")
		}
		return metrics.Counter(m.Delta.Int64), nil
	default:
	}
	return nil, fmt.Errorf("metric not implemented")
}

func (s *MemStorageDb) getId(ctx context.Context, id string) (metricsDB, error) {
	m := metricsDB{}
	fn := RetryQueryRowContext(s.db.QueryRowContext, 3, 1*time.Second)
	row := fn(ctx, queryGet, id)
	err := row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)

	return m, err
}

func (s *MemStorageDb) PutMetrics(ctx context.Context, m metrics.Metrics) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if m.Gauges != nil {
		err = s.putGauges(ctx, tx, m)
		if err != nil {
			return err
		}
	}

	if m.Counters != nil {
		err = s.putCounters(ctx, tx, m)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println("put metrics transaction failed - ", err)
		return err
	}
	return nil
}

func (s *MemStorageDb) putGauges(ctx context.Context, tx *sql.Tx, m metrics.Metrics) error {
	for id, value := range m.Gauges {
		fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
		result, err := fn(ctx, queryUpdateGauge, id, value)

		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
			_, err := fn(ctx, queryInsertGauge, id, value)

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *MemStorageDb) putCounters(ctx context.Context, tx *sql.Tx, m metrics.Metrics) error {
	for id, delta := range m.Counters {
		mdb := metricsDB{}
		fn := RetryQueryContext(s.db.QueryContext, 3, 1*time.Second)
		rows, err := fn(ctx, queryGet, id)

		if err != nil || rows.Err() != nil {
			return err
		}
		err = rows.Scan(&mdb.ID, &mdb.MType, &mdb.Value, &mdb.Delta)
		if err == sql.ErrNoRows {
			fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
			_, err := fn(ctx, queryInsertCounter, id, delta)

			if err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		v := metrics.Counter(0)
		if mdb.Delta.Valid {
			v = metrics.Counter(mdb.Delta.Int64)
		}
		hm := delta + v
		fn1 := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
		if _, err := fn1(ctx, queryUpdateCounter, id, hm); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemStorageDb) GetMetrics(parentCtx context.Context) (metrics.Metrics, error) {
	mcs := *metrics.New()
	rows, err := s.queryMetrics(parentCtx)
	if err != nil {
		return mcs, err
	}
	defer rows.Close()

	for rows.Next() {
		var m metricsDB
		err = rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
		if err != nil {
			return mcs, err
		}

		switch m.MType {
		case metrics.TypeGauge:
			if !m.Value.Valid {
				log.Println("NULL gauge value")
			}
			mcs.Gauges[metrics.Name(m.ID)] = metrics.Gauge(m.Value.Float64)
		case metrics.TypeCounter:
			if !m.Delta.Valid {
				log.Println("NULL counter value")
			}
			mcs.Counters[metrics.Name(m.ID)] = metrics.Counter(m.Delta.Int64)
		default:
			log.Println("not implemented metrics type")
		}
	}

	err = rows.Err()
	if err != nil {
		return mcs, err
	}

	return mcs, nil
}

func (s *MemStorageDb) queryMetrics(parentCtx context.Context) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	fn := RetryQueryContext(s.db.QueryContext, 3, 1*time.Second)
	return fn(ctx, queryGetMetrics)
}

func (s *MemStorageDb) Ping(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 1*time.Second)
	defer cancel()

	fn := Retry(s.db.PingContext, 3, 1*time.Second)
	err := fn(ctx)

	return err
}

func (s *MemStorageDb) Shutdown() error {
	err := s.db.Close()
	if err != nil {
		return err
	}

	log.Println("connection to database closed")
	return nil
}

func Retry(sender Sender, retries int, delay time.Duration) Sender {
	return func(ctx context.Context) error {
		for r := 0; ; r++ {
			err := sender(ctx)
			var pgErr *pgconn.PgError
			if !(errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)) || r >= retries {
				return err
			}

			log.Printf("Function call failed, retrying in %v", delay)

			delay = delay + time.Second*2

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func RetryQueryContext(sender QueryContext, retries int, delay time.Duration) QueryContext {
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		for r := 0; ; r++ {
			rows, err := sender(ctx, query, args...)
			var pgErr *pgconn.PgError
			if !(errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)) || r >= retries {
				return rows, err
			}

			log.Printf("Function call failed, retrying in %v", delay)

			delay = delay + time.Second*2

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return rows, err
			}
		}
	}
}

func RetryQueryRowContext(sender QueryRowContext, retries int, delay time.Duration) QueryRowContext {
	return func(ctx context.Context, query string, args ...any) *sql.Row {
		for r := 0; ; r++ {
			row := sender(ctx, query, args...)
			var pgErr *pgconn.PgError
			if !(errors.As(row.Err(), &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)) || r >= retries {
				return row
			}

			log.Printf("Function call failed, retrying in %v", delay)

			delay = delay + time.Second*2

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return row
			}
		}
	}
}

func RetryExecContext(sender ExecContext, retries int, delay time.Duration) ExecContext {
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		for r := 0; ; r++ {
			result, err := sender(ctx, query, args...)

			var pgErr *pgconn.PgError
			if !(errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)) || r >= retries {
				return result, err
			}

			log.Printf("Function call failed, retrying in %v", delay)

			delay = delay + time.Second*2

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return result, err
			}
		}
	}
}
