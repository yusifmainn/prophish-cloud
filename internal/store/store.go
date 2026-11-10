package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the data-access layer. All SQL lives here so handlers stay thin
// and query construction is centralized and auditable.
type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store { return &Store{db: db} }

// Context helper kept short for query call sites.
type ctx = context.Context
