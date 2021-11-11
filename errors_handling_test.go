package errors_handling

import (
	"log"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
)

func TestOne(t *testing.T) {
	var err error = &pgconn.PgError{
		Code:    "101",
		Message: "error message",
	}
	err = errors.WithStack(err)

	var err2 = new(pgconn.PgError)
	if errors.As(err, &err2) {
		log.Printf("database error with code %v: %+v", err2.Code, err)
	}
}

func one() error {
	return errors.New("error number one")
}

func TestDoubleWrappedErrors(t *testing.T) {
	err := errors.WithStack(one())
	log.Printf("%+v", err)
}
