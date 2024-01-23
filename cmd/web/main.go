package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/mabego/snippetbox-mysql/internal/models"
	"github.com/mabego/snippetbox-mysql/migrations"
)

const (
	AppPort         = ":4000"
	DBPort          = "3306"
	IdleTimeout     = time.Minute
	Path            = "sql"
	ReadTimeout     = 5 * time.Second
	SessionLifetime = 12 * time.Hour
	WriteTimeout    = 10 * time.Second
)

// config holds sensitive data from the environment variable "DSN"
type config struct {
	Dbname   string `json:"dbname"`
	Host     string `json:"host"`
	Password string `json:"password"`
	Username string `json:"username"`
}

type application struct {
	debug          bool
	errorLog       *log.Logger
	infoLog        *log.Logger
	snippets       models.SnippetModelInterface
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	env := os.Getenv("DSN")

	var cfg config
	if err := json.Unmarshal([]byte(env), &cfg); err != nil {
		errorLog.Fatal("error loading env: ", err)
	}

	_dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", cfg.Username, cfg.Password, cfg.Host, DBPort,
		cfg.Dbname)

	addr := flag.String("addr", AppPort, "HTTP network address")
	dsn := flag.String("dsn", _dsn, "Data source name")
	debug := flag.Bool("debug", false, "Enable debug mode in the browser")

	flag.Parse()

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			errorLog.Fatal(err)
		}
	}(db)

	if err := migration(db, cfg.Dbname, infoLog); err != nil {
		errorLog.Fatal(err)
	}

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = SessionLifetime

	app := &application{
		debug:          *debug,
		errorLog:       errorLog,
		infoLog:        infoLog,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.routes(),
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
		ErrorLog:     errorLog,
	}

	infoLog.Printf("Starting server on %s", *addr)
	errorLog.Fatal(srv.ListenAndServe())
}

// openDB wraps sql.Open and returns a sql.DB connection pool for a given data source name
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("database pool initialization: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection: %w", err)
	}

	return db, nil
}

func migration(db *sql.DB, dbname string, logger *log.Logger) error {
	embeddedSQLs, err := iofs.New(migrations.Migrations, Path)
	if err != nil {
		return fmt.Errorf("new migration iofs: %w", err)
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("migration database connection: %w", err)
	}

	m, err := migrate.NewWithInstance("migrations", embeddedSQLs, dbname, driver)
	if err != nil {
		return fmt.Errorf("new migration instance: %w", err)
	}

	if err := m.Up(); err != nil {
		// If the active migration version has not changed, for example, when new or additional containers run,
		// migrate.Up returns migrate.ErrNoChange.
		// If there is a match for this error, log it and return.
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Printf("migration: %s", err)
			return nil
		}
		return fmt.Errorf("migration: %w", err)
	}

	logger.Printf("migration complete")
	return nil
}
