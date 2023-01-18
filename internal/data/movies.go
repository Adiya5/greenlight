package data

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/shynggys9219/greenlight/internal/validator"
	"time"

	"github.com/lib/pq"
)

// By default, the keys in the JSON object are equal to the field names in the struct ( ID,
// CreatedAt, Title and so on).
type Movie struct {
	ID        int64     `json:"id"`                       // Unique integer ID for the movie
	CreatedAt time.Time `json:"-"`                        // Timestamp for when the movie is added to our database, "-" directive, hidden in response
	Title     string    `json:"title"`                    // Movie title
	Year      int32     `json:"year,omitempty"`           // Movie release year, "omitempty" - hide from response if empty
	Runtime   int32     `json:"runtime,omitempty,string"` // Movie runtime (in minutes), "string" - convert int to string
	Genres    []string  `json:"genres,omitempty"`         // Slice of genres for the movie (romance, comedy, etc.)
	Version   int32     `json:"version"`                  // The version number starts at 1 and will be incremented each
	// time the movie information is updated
}

type Director struct {
	ID      int64    `json:"id"` // Unique integer ID for the movie
	Name    string   `json:"name"`
	Surname string   `json:"surname"`
	Awards  []string `json:"awards,omitempty"` // Slice of genres for the movie (romance, comedy, etc.)

	// time the movie information is updated
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")
	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// Define a MovieModel struct type which wraps a sql.DB connection pool.
type MovieModel struct {
	DB *sql.DB
}

type DirectorsModel struct {
	DB *sql.DB
}

// method for inserting a new record in the movies table.
func (m MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies(title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`

	return m.DB.QueryRow(query, &movie.Title, &movie.Year, &movie.Runtime, pq.Array(&movie.Genres)).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}
func (m DirectorsModel) Insert(director *Director) error {
	query := `
	INSERT INTO directors(name, surname, awards)
	VALUES ($1, $2, $3)
	RETURNING id`

	return m.DB.QueryRow(query, &director.Name, &director.Surname, pq.Array(&director.Awards)).Scan(&director.ID)
}

// method for fetching a specific record from the movies table.
func (m MovieModel) Get(id int64) (*Movie, error) {
	query := `
SELECT id, created_at, title, year, runtime, genres, version
FROM movies
WHERE id = $1`
	var movie Movie
	m.DB.QueryRow(query, id).Scan(&movie.ID, &movie.CreatedAt, &movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version)
	return &movie, m.DB.QueryRow(query, id).Err()
}
func (m DirectorsModel) GetAll(name string, awards []string, filters Filters) ([]*Director, error) {
	// Update the SQL query to include the filter conditions.
	query := fmt.Sprintf(`
SELECT id, name, surname, awards
FROM directors
WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (awards @> $2 OR $2 = '{}')
ORDER BY %s %s, id ASC
LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortByAwards())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Pass the title and genres as the placeholder parameter values.
	args := []any{name, pq.Array(awards), filters.limit(), filters.offset()}
	// And then pass the args slice to QueryContext() as a variadic parameter.
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	directors := []*Director{}
	for rows.Next() {
		var director Director
		err := rows.Scan(
			&director.ID,
			&director.Name,
			&director.Surname,
			pq.Array(&director.Awards),
		)
		if err != nil {
			return nil, err
		}
		directors = append(directors, &director)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return directors, nil
}
func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, error) {
	// Update the SQL query to include the filter conditions.
	query := fmt.Sprintf(`
SELECT id, created_at, title, year, runtime, genres, version
FROM movies
WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (genres @> $2 OR $2 = '{}')
ORDER BY %s %s, id ASC
LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Pass the title and genres as the placeholder parameter values.
	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}
	// And then pass the args slice to QueryContext() as a variadic parameter.
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	movies := []*Movie{}
	for rows.Next() {
		var movie Movie
		err := rows.Scan(
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)
		if err != nil {
			return nil, err
		}
		movies = append(movies, &movie)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return movies, nil
}

// method for updating a specific record in the movies table.
func (m MovieModel) Update(movie *Movie) error {
	query := `
	UPDATE movies
	SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
	WHERE id = $5
	RETURNING version`

	return m.DB.QueryRow(query, movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID).Scan(
		&movie.Version)

}

// method for deleting a specific record from the movies table.
func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	// Construct the SQL query to delete the record.
	query := `
DELETE FROM movies
WHERE id = $1`

	result, err := m.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil

}
