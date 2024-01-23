package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const (
	PasswordHashCost = 12
)

type UserModelInterface interface {
	Insert(name, email, password string, owner bool) error
	Authenticate(email, password string) (int, error)
	Authorize(id int) (bool, error)
	Exists(id int) (bool, error)
	Get(id int) (*User, error)
	Owner() (bool, error)
	PasswordUpdate(id int, currentPassword, newPassword string) error
}

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
	Owner          bool
}

// UserModel wraps a database connection pool
type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string, owner bool) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	if err != nil {
		return err
	}

	statement := `INSERT INTO users (name, email, hashed_password, created, owner) VALUES (?, ?, ?, UTC_TIMESTAMP(), ?)`

	_, err = m.DB.Exec(statement, name, email, string(hashedPassword), owner)
	if err != nil {
		// Use errors.As to check if the error has the type *mysql.MySQLError.
		// If it does, the error is assigned to mySQLError and checked for error code 1062
		// and the constraint key users_uc_email.
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}
	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	query := `SELECT id, hashed_password FROM users WHERE email = ?`

	// Check if email exists in database.
	err := m.DB.QueryRow(query, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	// Check if the provided plain-text password matches the hashed password in the database.
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	return id, nil
}

func (m *UserModel) Authorize(id int) (bool, error) {
	var owner bool
	query := `SELECT owner FROM users WHERE id = ?`

	err := m.DB.QueryRow(query, id).Scan(&owner)
	if err != nil {
		return false, err
	}

	return owner, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT true FROM users WHERE id = ?)`

	err := m.DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (m *UserModel) Get(id int) (*User, error) {
	var user User

	query := `SELECT id, name, email, created FROM users WHERE id = ?`

	err := m.DB.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email, &user.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return &user, nil
}

func (m *UserModel) Owner() (bool, error) {
	var owner bool

	query := `SELECT EXISTS(SELECT true FROM users WHERE owner = true)`

	err := m.DB.QueryRow(query).Scan(&owner)
	if err != nil {
		return false, err
	}
	return owner, nil
}

func (m *UserModel) PasswordUpdate(id int, currentPassword, newPassword string) error {
	var currentHashedPassword []byte

	query := `SELECT hashed_password FROM users WHERE id = ?`

	err := m.DB.QueryRow(query, id).Scan(&currentHashedPassword)
	if err != nil {
		return err
	}

	// Check if the provided plain-text password matches the hashed password in the database.
	err = bcrypt.CompareHashAndPassword(currentHashedPassword, []byte(currentPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return err
	}

	// Hash the new password and update hashed_password column for the user
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), PasswordHashCost)
	if err != nil {
		return err
	}

	statement := `UPDATE users SET hashed_password = ? WHERE id = ?`

	_, err = m.DB.Exec(statement, string(newHashedPassword), id)
	return err
}
