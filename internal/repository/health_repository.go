package repository

import "database/sql"

type HealthRepository interface {
	Ping() error
}

type healthRepository struct {
	db *sql.DB
}

func NewHealthRepository(db *sql.DB) HealthRepository {
	return &healthRepository{db: db}
}

func (r *healthRepository) Ping() error {
	return r.db.Ping()
}
