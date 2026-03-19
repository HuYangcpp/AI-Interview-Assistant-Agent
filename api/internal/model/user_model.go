package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	RealName     string
	Phone        string
	Email        string
	Status       int16
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserModel struct {
	pool *pgxpool.Pool
}

func NewUserModel(pool *pgxpool.Pool) *UserModel {
	return &UserModel{pool: pool}
}

func (m *UserModel) FindById(ctx context.Context, id int64) (*User, error) {
	const query = `SELECT id, username, password_hash, role, real_name, phone, email, status, created_at, updated_at
FROM users WHERE id=$1`

	var u User
	err := m.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.Role,
		&u.RealName,
		&u.Phone,
		&u.Email,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (m *UserModel) FindByUsername(ctx context.Context, username string) (*User, error) {
	const query = `SELECT id, username, password_hash, role, real_name, phone, email, status, created_at, updated_at
FROM users WHERE username=$1`

	var u User
	err := m.pool.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.Role,
		&u.RealName,
		&u.Phone,
		&u.Email,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (m *UserModel) Insert(ctx context.Context, u *User) (int64, error) {
	const query = `INSERT INTO users (username, password_hash, role, real_name, phone, email, status)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query,
		u.Username,
		u.PasswordHash,
		u.Role,
		u.RealName,
		u.Phone,
		u.Email,
		u.Status,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *UserModel) UpdatePassword(ctx context.Context, userId int64, passwordHash string) error {
	const query = `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, userId, passwordHash)
	return err
}

func (m *UserModel) Update(ctx context.Context, u *User) error {
	const query = `UPDATE users
SET real_name=$2, phone=$3, email=$4, status=$5, updated_at=NOW()
WHERE id=$1`

	_, err := m.pool.Exec(ctx, query, u.ID, u.RealName, u.Phone, u.Email, u.Status)
	return err
}

func (m *UserModel) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM users WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, id)
	return err
}
