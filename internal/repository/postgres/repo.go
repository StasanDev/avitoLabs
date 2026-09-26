package repository

import "github.com/jackc/pgx/v5/pgxpool"

type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
	}
}
