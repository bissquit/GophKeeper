// CLI for the GophKeeper password manager
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bissquit/gophkeeper/internal/client/api"
	"github.com/bissquit/gophkeeper/internal/client/crypto"
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

auth:
  version                          print client version and build date
  ping                             check server health
  register <login>                 register a new user
  login <login>                    log in as an existing user
  logout                           clear the local session
  whoami                           show the currently logged-in user

secrets (require login; payload is encrypted client-side):
  add credentials NAME [--login L] [--password P] [--meta M]
  add text NAME --text T [--meta M]
  add binary NAME --file F [--meta M]
  add card NAME --number N --expiry E [--cvv C] [--holder H] [--meta M]
  list                             list every secret version (no decryption)
  get NAME                         decrypt and show the latest version named NAME
  delete NAME                      remove every version of the secret named NAME

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
	case "add":
		err = cmdAdd()
	case "list":
		err = cmdList()
	case "get":
		err = cmdGet()
	case "delete":
		err = cmdDelete()
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

// secret payload shapes — each is JSON-serialized, then encrypted

type credentialsPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type textPayload struct {
	Text string `json:"text"`
}

type binaryPayload struct {
	Data []byte `json:"data"`
}

type cardPayload struct {
	Number string `json:"number"`
	Expiry string `json:"expiry"`
	CVV    string `json:"cvv,omitempty"`
	Holder string `json:"holder,omitempty"`
}

func cmdAdd() error {
	if len(os.Args) < 4 {
		return errors.New("usage: gophkeeper add <credentials|text|binary|card> NAME [flags]")
	}
	kind := os.Args[2]
	name := os.Args[3]
	rest := os.Args[4:]

	sess, key, err := openSession()
	if err != nil {
		return err
	}

	var (
		payload []byte
		meta    string
	)

	switch kind {
	case "credentials":
		fs := flag.NewFlagSet("add credentials", flag.ContinueOnError)
		login := fs.String("login", "", "account login")
		password := fs.String("password", "", "account password (empty -> prompt)")
		fs.StringVar(&meta, "meta", "", "free-form meta")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *password == "" {
			p, err := readPasswordPrompt("Secret password: ")
			if err != nil {
				return err
			}
			*password = p
		}
		payload, err = json.Marshal(credentialsPayload{Login: *login, Password: *password})
	case "text":
		fs := flag.NewFlagSet("add text", flag.ContinueOnError)
		text := fs.String("text", "", "inline text")
		fs.StringVar(&meta, "meta", "", "free-form meta")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		payload, err = json.Marshal(textPayload{Text: *text})
	case "binary":
		fs := flag.NewFlagSet("add binary", flag.ContinueOnError)
		file := fs.String("file", "", "read bytes from file ('-' for stdin)")
		fs.StringVar(&meta, "meta", "", "free-form meta")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *file == "" {
			return errors.New("--file is required for binary")
		}
		b, err := readFileOrStdin(*file)
		if err != nil {
			return err
		}
		payload, err = json.Marshal(binaryPayload{Data: b})
		if err != nil {
			return err
		}
	case "card":
		fs := flag.NewFlagSet("add card", flag.ContinueOnError)
		number := fs.String("number", "", "card number")
		expiry := fs.String("expiry", "", "MM/YY")
		cvv := fs.String("cvv", "", "cvv")
		holder := fs.String("holder", "", "card holder name")
		fs.StringVar(&meta, "meta", "", "free-form meta")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *number == "" || *expiry == "" {
			return errors.New("--number and --expiry are required for card")
		}
		payload, err = json.Marshal(cardPayload{Number: *number, Expiry: *expiry, CVV: *cvv, Holder: *holder})
	default:
		return fmt.Errorf("unknown secret type %q", kind)
	}
	if err != nil {
		return err
	}

	ciphertext, err := crypto.Encrypt(key, payload)
	if err != nil {
		return err
	}
	sec, err := api.New(sess.Server, sess.Token).Create(context.Background(), kind, name, ciphertext, meta)
	if err != nil {
		return err
	}
	fmt.Printf("stored %s %q (id=%s, version=%d)\n", kind, name, sec.ID, sec.Version)
	return nil
}

func cmdList() error {
	sess, err := session.Load()
	if err != nil {
		return err
	}
	items, err := api.New(sess.Server, sess.Token).List(context.Background())
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("no secrets")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tVERSION\tUPDATED\tID")
	for _, s := range items {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\n", s.Name, s.Type, s.Version, s.UpdatedAt.Format("2006-01-02 15:04:05"), s.ID)
	}
	return tw.Flush()
}

func cmdGet() error {
	if len(os.Args) < 3 {
		return errors.New("usage: gophkeeper get NAME")
	}
	name := os.Args[2]
	sess, key, err := openSession()
	if err != nil {
		return err
	}
	client := api.New(sess.Server, sess.Token)
	items, err := client.List(context.Background())
	if err != nil {
		return err
	}
	latest, ok := latestByName(items, name)
	if !ok {
		return fmt.Errorf("no secret named %q", name)
	}
	plaintext, err := crypto.Decrypt(key, latest.Data)
	if err != nil {
		return fmt.Errorf("decrypt: %w (wrong master password?)", err)
	}
	return printPayload(latest, plaintext)
}

func cmdDelete() error {
	if len(os.Args) < 3 {
		return errors.New("usage: gophkeeper delete NAME")
	}
	name := os.Args[2]
	sess, err := session.Load()
	if err != nil {
		return err
	}
	client := api.New(sess.Server, sess.Token)
	items, err := client.List(context.Background())
	if err != nil {
		return err
	}
	latest, ok := latestByName(items, name)
	if !ok {
		return fmt.Errorf("no secret named %q", name)
	}
	if err := client.Delete(context.Background(), latest.ID); err != nil {
		return err
	}
	fmt.Printf("deleted %s %q (id=%s)\n", latest.Type, name, latest.ID)
	return nil
}

// latestByName returns the highest-version Secret with the given name
func latestByName(items []api.Secret, name string) (api.Secret, bool) {
	var (
		best  api.Secret
		found bool
	)
	for _, s := range items {
		if s.Name == name && (!found || s.Version > best.Version) {
			best, found = s, true
		}
	}
	return best, found
}

func printPayload(s api.Secret, plaintext []byte) error {
	fmt.Printf("name: %s\ntype: %s\nversion: %d\nupdated: %s\n",
		s.Name, s.Type, s.Version, s.UpdatedAt.Format("2006-01-02 15:04:05"))
	if s.Meta != "" {
		fmt.Printf("meta: %s\n", s.Meta)
	}
	switch s.Type {
	case "credentials":
		var p credentialsPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		fmt.Printf("login: %s\npassword: %s\n", p.Login, p.Password)
	case "text":
		var p textPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		fmt.Printf("text:\n%s\n", p.Text)
	case "binary":
		var p binaryPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		_, err := os.Stdout.Write(p.Data)
		return err
	case "card":
		var p cardPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		fmt.Printf("number: %s\nexpiry: %s\n", p.Number, p.Expiry)
		if p.CVV != "" {
			fmt.Printf("cvv: %s\n", p.CVV)
		}
		if p.Holder != "" {
			fmt.Printf("holder: %s\n", p.Holder)
		}
	default:
		fmt.Printf("data (hex/utf8):\n%s\n", string(plaintext))
	}
	return nil
}

// openSession loads the session and prompts for the master password,
// returning the derived AES key alongside
func openSession() (*session.Session, []byte, error) {
	sess, err := session.Load()
	if err != nil {
		return nil, nil, err
	}
	mp, err := readPasswordPrompt("Master password: ")
	if err != nil {
		return nil, nil, err
	}
	return sess, crypto.DeriveKey(mp, sess.Login), nil
}

func readFileOrStdin(path string) ([]byte, error) {
	if path == "-" {
		if stdinReader != nil {
			return io.ReadAll(stdinReader)
		}
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func readPassword() (string, error) {
	return readPasswordPrompt("Password: ")
}

// stdinReader is shared across prompts so successive non-TTY reads
// see consecutive lines instead of an empty buffer
var stdinReader *bufio.Reader

func readPasswordPrompt(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	// don't disable echo if not tty (e.g. file or pipe)
	if !term.IsTerminal(fd) {
		if stdinReader == nil {
			stdinReader = bufio.NewReader(os.Stdin)
		}
		line, err := stdinReader.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	fmt.Print(prompt)
	b, err := term.ReadPassword(fd)
	fmt.Println()
	return string(b), err
}
