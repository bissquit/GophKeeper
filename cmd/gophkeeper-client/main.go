// CLI for the GophKeeper password manager
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bissquit/gophkeeper/internal/client/api"
	"github.com/bissquit/gophkeeper/internal/client/session"
	"golang.org/x/term"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func serverURL() string {
	if s := os.Getenv("GOPHKEEPER_SERVER"); s != "" {
		return s
	}
	return "http://localhost:8080"
}

const usageText = `usage: gophkeeper <command> [args]

commands:
  version                 print client version and build date
  ping                    check server health
  register <login>        register a new user
  login <login>           log in as an existing user
  logout                  clear the local session
  whoami                  show the currently logged-in user

env: GOPHKEEPER_SERVER (default http://localhost:8080)`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usageText)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "version":
		fmt.Printf("gophkeeper %s (built %s)\n", version, buildDate)
	case "ping":
		err = api.New(serverURL(), "").Ping(context.Background())
		if err == nil {
			fmt.Println("ok")
		}
	case "register":
		err = cmdRegister()
	case "login":
		err = cmdLogin()
	case "logout":
		if err = session.Clear(); err == nil {
			fmt.Println("logged out")
		}
	case "whoami":
		err = cmdWhoami()
	default:
		fmt.Println(usageText)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func cmdRegister() error {
	if len(os.Args) < 3 {
		return errors.New("usage: gophkeeper register <login>")
	}
	login := os.Args[2]
	password, err := readPassword()
	if err != nil {
		return err
	}
	token, err := api.New(serverURL(), "").Register(context.Background(), login, password)
	if err != nil {
		if errors.Is(err, api.ErrConflict) {
			return fmt.Errorf("login %q is already taken", login)
		}
		return err
	}
	if err := session.Save(&session.Session{Server: serverURL(), Login: login, Token: token}); err != nil {
		return err
	}
	fmt.Println("registered and logged in as", login)
	return nil
}

func cmdLogin() error {
	if len(os.Args) < 3 {
		return errors.New("usage: gophkeeper login <login>")
	}
	login := os.Args[2]
	password, err := readPassword()
	if err != nil {
		return err
	}
	token, err := api.New(serverURL(), "").Login(context.Background(), login, password)
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			return errors.New("invalid login or password")
		}
		return err
	}
	if err := session.Save(&session.Session{Server: serverURL(), Login: login, Token: token}); err != nil {
		return err
	}
	fmt.Println("logged in as", login)
	return nil
}

func cmdWhoami() error {
	s, err := session.Load()
	if err != nil {
		return err
	}
	fmt.Printf("login: %s\nserver: %s\n", s.Login, s.Server)
	return nil
}

func readPassword() (string, error) {
	fmt.Print("Password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(b), err
}
