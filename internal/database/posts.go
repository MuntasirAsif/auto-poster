package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	ID          int64
	Platform    string
	Content     string
	Status      string
	ScheduledAt *time.Time
	PublishedAt *time.Time
	Error       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PostRepository struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{pool: pool}
}

func (r *PostRepository) Create(ctx context.Context, platform, content string, scheduledAt *time.Time) (*Post, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO posts (platform, content, scheduled_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, platform, content, status, scheduled_at, published_at, error, created_at, updated_at`,
		platform, content, scheduledAt,
	)

	return scanPost(row)
}

func (r *PostRepository) GetByID(ctx context.Context, id int64) (*Post, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, platform, content, status, scheduled_at, published_at, error, created_at, updated_at
		 FROM posts WHERE id = $1`,
		id,
	)

	return scanPost(row)
}

func (r *PostRepository) List(ctx context.Context) ([]Post, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, platform, content, status, scheduled_at, published_at, error, created_at, updated_at
		 FROM posts ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Platform, &p.Content, &p.Status, &p.ScheduledAt, &p.PublishedAt, &p.Error, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

func (r *PostRepository) UpdateStatus(ctx context.Context, id int64, status string, publishedAt *time.Time, errMsg *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE posts SET status = $2, published_at = $3, error = $4, updated_at = now()
		 WHERE id = $1`,
		id, status, publishedAt, errMsg,
	)
	return err
}

func (r *PostRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM posts WHERE id = $1`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (*Post, error) {
	var p Post
	err := row.Scan(&p.ID, &p.Platform, &p.Content, &p.Status, &p.ScheduledAt, &p.PublishedAt, &p.Error, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}