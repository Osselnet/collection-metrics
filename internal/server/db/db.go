package db

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"time"
)

type DBStorage interface {
	Ping() error
	Shutdown() error
}

type Storage struct {
	db  *sql.DB
	dsn string
}

func New(dsn string) DBStorage {
	s := &Storage{
		dsn: dsn,
	}
	err := s.init()
	if err != nil {
		log.Printf("Database initialization error: %v", err)
	}
	return s
}

func (s *Storage) init() error {
	db, err := sql.Open("pgx", s.dsn)
	if err != nil {
		return err
	}
	s.db = db

	return nil
}

func (s *Storage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Shutdown() error {
	err := s.db.Close()
	if err != nil {
		return err
	}

	log.Println("connection to database closed")
	return nil
}
