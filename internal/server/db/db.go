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

	queryTableValidation = `select * from metrics`
	queryCreateTable     = `
		CREATE TABLE public.metrics (
			id text NOT NULL,
			type text NOT NULL,
			value double precision,
			delta bigint,
			PRIMARY KEY (id)
		);
	`
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

type MemStorage struct {
	db  *sql.DB
	dsn string
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

func New(dsn string) DateBaseStorage {
	s := &MemStorage{
		dsn: dsn,
	}
	err := s.init()
	if err != nil {
		log.Printf("Database initialization error: %v", err)
	}
	return s
}

func (s *MemStorage) init() error {
	db, err := sql.Open("pgx", s.dsn)
	if err != nil {
		return err
	}
	s.db = db

	ctx, cancel := context.WithTimeout(context.Background(), initTimeOut)
	defer cancel()

	fn := RetryExecContext(db.ExecContext, 3, 1*time.Second)
	_, err = fn(ctx, queryTableValidation)

	if err != nil {
		fn := RetryExecContext(db.ExecContext, 3, 1*time.Second)
		_, err = fn(ctx, queryCreateTable)

		if err != nil {
			return err
		}

		log.Println("table `metrics` created")
	}
	return nil
}

func (s *MemStorage) Put(parentCtx context.Context, id string, val interface{}) error {
	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	switch m := val.(type) {
	case metrics.Gauge:
		fn := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
		result, err := fn(ctx, queryUpdateGauge, id, m)

		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			fn := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
			_, err = fn(ctx, queryInsertGauge, id, m)

			if err != nil {
				return err
			}
		}
	case metrics.Counter:
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
			_, err = fn1(ctx, queryInsertCounter, id, m)

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

		count := m + metrics.Counter(mdb.Delta.Int64)

		fn2 := RetryExecContext(s.db.ExecContext, 3, 1*time.Second)
		_, err = fn2(ctx, queryUpdateCounter, id, count)

		if err != nil {
			return err
		}
	default:
		return errors.New("storage: metric not implemented")
	}
	return nil
}

func (s *MemStorage) Get(parentCtx context.Context, id string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	m := metricsDB{}
	fn := RetryQueryRowContext(s.db.QueryRowContext, 3, 1*time.Second)
	row := fn(ctx, queryGet, id)
	err := row.Scan(&m.ID, &m.MType, &m.Value, &m.Delta)
	if err != nil {
		return nil, err
	}

	switch m.MType {
	case "gauge":
		if !m.Value.Valid {
			return nil, fmt.Errorf("NULL gauge value")
		}
		return metrics.Gauge(m.Value.Float64), nil
	case "counter":
		if !m.Delta.Valid {
			return nil, fmt.Errorf("NULL counter value")
		}
		return metrics.Counter(m.Delta.Int64), nil
	default:
	}

	return nil, fmt.Errorf("metric not implemented")
}

func (s *MemStorage) PutMetrics(ctx context.Context, m metrics.Metrics) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if m.Gauges != nil {
		for id, value := range m.Gauges {
			fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
			result, err := fn(ctx, "UPDATE metrics SET value = $2 WHERE id = $1", id, value)

			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if count == 0 {
				fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
				_, err := fn(ctx, "INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)", id, value)

				if err != nil {
					return err
				}
			}
		}
	}

	if m.Counters != nil {
		for id, delta := range m.Counters {
			mdb := metricsDB{}
			fn := RetryQueryContext(s.db.QueryContext, 3, 1*time.Second)
			rows, err := fn(ctx, "SELECT id, type, value, delta FROM metrics WHERE id=$1", id)
			if err != nil {
				return err
			}
			err = rows.Scan(&mdb.ID, &mdb.MType, &mdb.Value, &mdb.Delta)
			if err == sql.ErrNoRows {
				fn := RetryExecContext(tx.ExecContext, 3, 1*time.Second)
				_, err := fn(ctx, "INSERT INTO metrics (id, type, delta) VALUES ($1, 'counter', $2)", id, delta)

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
			if _, err := fn1(ctx, "UPDATE metrics SET delta = $2 WHERE id = $1", id, hm); err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println("put metrics transaction failed - ", err)
		return err
	}
	return nil
}

func (s *MemStorage) GetMetrics(parentCtx context.Context) (metrics.Metrics, error) {
	mcs := *metrics.New()

	ctx, cancel := context.WithTimeout(parentCtx, queryTimeOut)
	defer cancel()

	fn := RetryQueryContext(s.db.QueryContext, 3, 1*time.Second)
	rows, err := fn(ctx, queryGetMetrics)

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
		case "gauge":
			if !m.Value.Valid {
				log.Println("NULL gauge value")
			}
			mcs.Gauges[metrics.Name(m.ID)] = metrics.Gauge(m.Value.Float64)
		case "counter":
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

func (s *MemStorage) Ping(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 1*time.Second)
	defer cancel()

	fn := Retry(s.db.PingContext, 3, 1*time.Second)
	err := fn(ctx)

	return err
}

func (s *MemStorage) Shutdown() error {
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
