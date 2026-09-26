package main

import "testing"

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"env":"dev","db_host":"db","db_user":"platform","db_password":"secret","db_name":"platform"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" || cfg.DBPort != 5432 {
		t.Fatalf("defaults: listen=%s port=%d", cfg.Listen, cfg.DBPort)
	}
}

func TestParseConfigRequiresIdentity(t *testing.T) {
	if _, err := ParseConfig([]byte(`{"db_host":"db"}`)); err == nil {
		t.Fatal("expected missing env to fail")
	}
}

func TestDSNEscapesPassword(t *testing.T) {
	cfg := Config{
		Env:        "dev",
		DBHost:     "db",
		DBPort:     5432,
		DBUser:     "platform",
		DBPassword: "a b",
		DBName:     "platform",
	}
	got := cfg.DSN()
	if got != "postgres://platform:a%20b@db:5432/platform?sslmode=disable" {
		t.Fatalf("dsn: %s", got)
	}
}
