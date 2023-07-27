package db

import (
	"context"
	"database/sql"
	"log"
	"time"
)

const (
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
)

func New(dsn string) DateBaseStorage {
	s := &MemStorageDb{}
	err := s.init(dsn)
	if err != nil {
		log.Printf("Database initialization error: %v", err)
	}
	return s
}

func (s *MemStorageDb) init(dsn string) error {
	db, err := sql.Open("pgx", dsn)
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
