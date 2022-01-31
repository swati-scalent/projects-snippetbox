package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golangcollege/sessions"
	"github.com/projects/snippetbox/pkg/models"
	"github.com/projects/snippetbox/pkg/models/mysql"
)

type contextKey string

var contextKeyUser = contextKey("user")

// By injecting dependencies into your handlers, makes the code more explicit, less error-prone and
//easier to unit test.
//To inject dependencies is to put them into a custom application struct, and
//then define your handler functions as methods against application.
/*type application struct {
	errorLog      *log.Logger                   //A Logger represents an active logging object
	infoLog       *log.Logger                   //that generates lines of output to an io.Writer.
	session       *sessions.Session             //
	snippets      *mysql.SnippetModel           //makes the SnippetModel object available to our handlers.
	templateCache map[string]*template.Template //Added a templateCache field
	users         *mysql.UserModel
}*/

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
	session  *sessions.Session
	snippets interface {
		Insert(string, string, string) (int, error)
		Get(int) (*models.Snippet, error)
		Latest() ([]*models.Snippet, error)
	}
	templateCache map[string]*template.Template
	users         interface {
		Insert(string, string, string) error
		Authenticate(string, string) (int, error)
		Get(int) (*models.User, error)
	}
}

func main() {
	//String() defines a string flag with specified name, default value, and usage string.
	//The return value is the address of a string variable that stores the value of the flag.

	addr := flag.String("addr", ":8090", "HTTP network address, (e.g. :8090)")

	dsn := flag.String("dsn", "web:Vicky@123@/snippetbox?parseTime=true", "MySQL data source name")

	//defining a 32-byte secret key to encrypt and authenticate the session data
	secret := flag.String("secret", "s6Ndh+pPbnzHbS*+9Pk8qGWhTzbpa@ge", "Secret key")

	//Parse() parses the command-line flags from os.Args[1:].
	//Must be called after all flags are defined and before flags are accessed by the program.

	//The parseTime=true part of the DSN above is a driver-specific parameter which instructs
	//driver to convert SQL TIME and DATE fields to Go time.Time objects.

	flag.Parse()

	//log.New() creates a new Logger.
	//The first variable sets the destination to which log data will be written.
	//The prefix appears at the beginning of each generated log line.
	//The last argument defines the logging properties.

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)

	if err != nil {
		errorLog.Fatal(err)
	}

	defer db.Close()

	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		errorLog.Fatal(err)
	}

	// sessions.New() function initializes a new session manager, passing in the secret key as the
	//parameter. Then configuring it so sessions always expires after 12 hours.
	session := sessions.New([]byte(*secret))
	session.Lifetime = 12 * time.Hour

	//app is variable of type struct

	app := &application{
		errorLog:      errorLog,
		infoLog:       infoLog,
		session:       session,
		snippets:      &mysql.SnippetModel{DB: db},
		templateCache: templateCache,
		users:         &mysql.UserModel{DB: db},
	}

	// Initializing a tls.Config struct to hold the non-default TLS settings we want the server to use.
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	//A http.Server defines parameters for running an HTTP server.
	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(), //Initializing a mysql.SnippetModel instance
		TLSConfig:    tlsConfig,    //Set the server's TLSConfig field to use the tlsConfig variable
		IdleTimeout:  time.Minute,  //Adding Idle, Read and Write timeouts to the server
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)

	// ListenAndServeTLS() method to start the HTTPS server and passing paths
	// and corresponding private key as parameters to the TLS certificate
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

// The openDB() function wraps sql.Open() and returns a sql.DB connection pool for a given DSN.
func openDB(dsn string) (*sql.DB, error) {

	//The sql.Open() function doesn’t actually create any connections, all it does is initialize the pool for future use

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	//db.Ping() method to create a connection and check for any errors.

	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
