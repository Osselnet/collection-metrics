package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
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

	Ping() error
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

	_, err = db.ExecContext(ctx, queryTableValidation)
	if err != nil {
		_, err = db.ExecContext(ctx, queryCreateTable)
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
		result, err := s.db.ExecContext(ctx, queryUpdateGauge, id, m)
		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			_, err = s.db.ExecContext(ctx, queryInsertGauge, id, m)
			if err != nil {
				return err
			}
		}
	case metrics.Counter:
		// запишем новое значение счётчика
		result, err := s.db.ExecContext(ctx, queryGet, id)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			_, err = s.db.ExecContext(ctx, queryInsertCounter, id, m)
			if err != nil {
				return err
			}
			return nil
		}

		mdb := metricsDB{}
		row := s.db.QueryRowContext(ctx, queryGet, id)
		err = row.Scan(&mdb.ID, &mdb.MType, &mdb.Value, &mdb.Delta)
		if err != nil {
			return err
		}

		if !mdb.Delta.Valid {
			mdb.Delta.Int64 = 0
		}

		count := m + metrics.Counter(mdb.Delta.Int64)

		_, err = s.db.ExecContext(ctx, queryUpdateCounter, id, count)
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
	row := s.db.QueryRowContext(ctx, queryGet, id)
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
			result, err := tx.ExecContext(ctx, "UPDATE metrics SET value = $2 WHERE id = $1", id, value)
			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if count == 0 {
				_, err := tx.ExecContext(ctx, "INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)", id, value)
				if err != nil {
					return err
				}
			}
		}
	}

	if m.Counters != nil {
		for id, delta := range m.Counters {
			mdb := metricsDB{}
			row := tx.QueryRowContext(ctx, "SELECT id, type, value, delta FROM metrics WHERE id=$1", id)
			err = row.Scan(&mdb.ID, &mdb.MType, &mdb.Value, &mdb.Delta)
			if err == sql.ErrNoRows {
				_, err = tx.ExecContext(ctx, "INSERT INTO metrics (id, type, delta) VALUES ($1, 'counter', $2)", id, delta)
				if err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}

			// запишим увеличенное значение
			v := metrics.Counter(0)
			if mdb.Delta.Valid {
				v = metrics.Counter(mdb.Delta.Int64)
			}
			hm := delta + v
			if _, err = tx.ExecContext(ctx, "UPDATE metrics SET delta = $2 WHERE id = $1", id, hm); err != nil {
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

	rows, err := s.db.QueryContext(ctx, queryGetMetrics)
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

func (s *MemStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *MemStorage) Shutdown() error {
	err := s.db.Close()
	if err != nil {
		return err
	}

	log.Println("connection to database closed")
	return nil
}
