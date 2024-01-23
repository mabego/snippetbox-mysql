package models

import (
	"database/sql"
	"fmt"
)

type ReviewModelInterface interface {
	Insert(userID, snippetID int) error
	Exists(userID, snippetID int) (bool, error)
	Get(userID, snippetID int) (*Review, error)
	Update(userID, snippetID int) error
}

type Review struct {
	UserID    int
	SnippetID int
	Reviews   uint8
}

// ReviewModel wraps a database connection pool
type ReviewModel struct {
	DB *sql.DB
}

func (m *ReviewModel) Insert(userID, snippetID int) error {
	statement := `
		INSERT INTO reviews (userID, snippetID) 
		SELECT users.id, snippets.id
		FROM users, snippets 
		WHERE users.id = ? AND snippets.id = ?`

	// Exec inserts a row for specific snippet and user into the Reviews table.
	// The default value for Reviews is zero. See migrations/sql/000004_create_reviews_table.up.sql.
	_, err := m.DB.Exec(statement, userID, snippetID)
	if err != nil {
		return err
	}
	return nil
}

func (m *ReviewModel) Exists(userID, snippetID int) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT true FROM reviews WHERE userID = ? AND snippetID = ?)`

	err := m.DB.QueryRow(query, userID, snippetID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("row scan error: %w", err)
	}
	return exists, nil
}

func (m *ReviewModel) Get(userID, snippetID int) (*Review, error) {
	// Create a row on the first visit and return; the default for reviews is 0.
	exists, err := m.Exists(userID, snippetID)
	if err != nil {
		return nil, err
	}
	if !exists {
		err := m.Insert(userID, snippetID)
		if err != nil {
			return nil, err
		}
		return &Review{
			UserID:    userID,
			SnippetID: snippetID,
			Reviews:   0,
		}, nil
	}

	review := &Review{}

	statement := `SELECT review FROM reviews WHERE userID = ? AND snippetID = ?`

	err = m.DB.QueryRow(statement, userID, snippetID).Scan(&review.Reviews)
	if err != nil {
		return nil, err
	}

	return review, nil
}

func (m *ReviewModel) Update(userID, snippetID int) error {
	// Start a transaction to lock the row for update for multiple reviewers using the same login.
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}

	lockRow, err := tx.Prepare("SELECT review FROM reviews WHERE userID = ? AND snippetID = ? FOR UPDATE")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer lockRow.Close()
	if _, err := lockRow.Exec(userID, snippetID); err != nil {
		tx.Rollback()
		return err
	}

	updateRow, err := tx.Prepare("UPDATE reviews SET review = review + 1 WHERE userID = ? AND snippetID = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer updateRow.Close()
	if _, err := updateRow.Exec(userID, snippetID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
