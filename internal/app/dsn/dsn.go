package dsn

import (
	"fmt"
	"os"
)

func FromEnv() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "battery"
	}
	pass := os.Getenv("DB_PASS")
	if pass == "" {
		pass = "battery_lab"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "battery_life"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbname)
}
