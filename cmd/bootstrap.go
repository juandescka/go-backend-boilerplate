package cmd

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"hris-core-service/internal/infrastructure/seeding"
	"hris-core-service/internal/shared/password"
)

func newBootstrapCommand() *cobra.Command {
	bootstrap := &cobra.Command{Use: "bootstrap", Short: "Initialize a local platform administrator"}
	var expected string
	admin := &cobra.Command{
		Use: "super-admin", Short: "Create the first local platform administrator", Args: cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tty, ok := cmd.InOrStdin().(*os.File)
			if !ok || !term.IsTerminal(int(tty.Fd())) {
				return errors.New("administrator bootstrap requires an interactive terminal")
			}
			target, err := seeding.Open(cmd.Context(), expected)
			if err != nil {
				return err
			}
			defer target.Close()

			email, name, hash, err := readAdminAccount(tty, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if err := target.Write(cmd.Context(), func(ctx context.Context, tx *sql.Tx) error {
				return seeding.BootstrapSuperAdmin(ctx, tx, email, name, hash)
			}); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Initial platform administrator created.")
			return nil
		},
	}
	admin.Flags().StringVar(&expected, "target", "", "Confirm the effective PostgreSQL host:port/database")
	bootstrap.AddCommand(admin)
	return bootstrap
}

func readAdminAccount(tty *os.File, out io.Writer) (email, name, hash string, err error) {
	input := bufio.NewReaderSize(tty, 4096)
	readLine := func(prompt string) (string, error) {
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return "", errors.New("could not write administrator prompt")
		}
		line, err := input.ReadSlice('\n')
		if err != nil {
			return "", errors.New("could not read administrator identity")
		}
		return strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r"), nil
	}
	email, err = readLine("Administrator email: ")
	if err != nil {
		return "", "", "", err
	}
	email = strings.Trim(email, " ")
	if len(email) > 255 || !utf8.ValidString(email) || validator.New().Var(email, "required,email") != nil {
		return "", "", "", errors.New("enter a valid administrator email")
	}
	name, err = readLine("Administrator display name: ")
	if err != nil {
		return "", "", "", err
	}
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 150 || strings.ContainsFunc(name, unicode.IsControl) {
		return "", "", "", errors.New("enter a display name of 1 to 150 characters")
	}
	// Refuse type-ahead instead of accepting a password from buffered input.
	if input.Buffered() != 0 {
		return "", "", "", errors.New("enter each value only after its terminal prompt")
	}
	secret, err := readHidden(tty, out, "New password (input hidden): ")
	if err != nil {
		return "", "", "", err
	}
	defer clear(secret)
	if !validBootstrapPassword(secret) {
		return "", "", "", errors.New("password must contain at least 15 characters and at most 1024 bytes of valid UTF-8")
	}
	confirmation, err := readHidden(tty, out, "Confirm password (input hidden): ")
	if err != nil {
		return "", "", "", err
	}
	defer clear(confirmation)
	if !bytes.Equal(secret, confirmation) {
		return "", "", "", errors.New("password confirmation did not match")
	}
	hash, err = password.Hash(string(secret))
	return email, name, hash, err
}

func validBootstrapPassword(secret []byte) bool {
	return password.ValidInput(string(secret)) && utf8.RuneCount(secret) >= 15
}

func readHidden(tty *os.File, out io.Writer, prompt string) ([]byte, error) {
	fd := int(tty.Fd())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	// Suppress echo before showing the prompt so fast input cannot be echoed.
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, errors.New("could not disable terminal echo")
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case sig := <-signals:
			// Default process exit would skip defers while the read is blocked.
			_ = term.Restore(fd, state)
			os.Exit(128 + int(sig.(syscall.Signal)))
		case <-done:
			select {
			case sig := <-signals:
				os.Exit(128 + int(sig.(syscall.Signal)))
			default:
			}
		}
	}()
	defer func() {
		_ = term.Restore(fd, state)
		signal.Stop(signals)
		close(done)
		<-stopped
	}()
	// The password reader edits the input without changing terminal settings.
	// The outer cleanup and interrupt handler both restore the same state.
	terminal := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{tty, out}, "")
	value, err := terminal.ReadPassword(prompt)
	if err != nil {
		return nil, errors.New("could not read hidden password from the terminal")
	}
	return []byte(value), nil
}

func init() { RootCmd.AddCommand(newBootstrapCommand()) }
