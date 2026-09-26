package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"platform-lab/internal/lab"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	command := os.Args[1]
	env, err := parseEnv(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(2)
	}
	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := lab.Options{Root: root, Env: env}
	switch command {
	case "up":
		err = opts.Up(ctx)
	case "converge":
		err = opts.Converge(ctx)
	case "smoke":
		err = opts.Smoke(ctx)
	case "down":
		err = opts.Down(ctx)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "platform %s: %v\n", command, err)
		os.Exit(1)
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func parseEnv(args []string) (string, error) {
	fs := flag.NewFlagSet("platform", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	env := fs.String("env", "dev", "environment: dev or prod")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if *env != "dev" && *env != "prod" {
		return "", fmt.Errorf("-env must be dev or prod")
	}
	return *env, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: platform <up|converge|smoke|down> [-env dev|prod]

  up        build the base image and apply Terraform
  converge  build the server, render inventory, and run Ansible
  smoke     apply, prove idempotence, configure, and check /health and /
  down      destroy the environment
`)
}
