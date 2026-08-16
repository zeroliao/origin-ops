package usercli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"strings"

	"origin-ops/internal/authn"
	"origin-ops/internal/config"
)

func Run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("user action is required: add, set-password, enable, disable, or list")
	}
	action := args[0]
	flags := flag.NewFlagSet("origin-ops user "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "configuration file")
	username := flags.String("username", "", "username")
	passwordStdin := flags.Bool("password-stdin", false, "read the password from standard input")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	store := authn.NewStore(cfg.AuthFile)

	switch action {
	case "add":
		password, err := readPassword(stdin, *passwordStdin)
		if err != nil {
			return err
		}
		if err := store.Add(*username, password); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "user %s added\n", *username)
	case "set-password":
		password, err := readPassword(stdin, *passwordStdin)
		if err != nil {
			return err
		}
		if err := store.SetPassword(*username, password); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "password updated for %s\n", *username)
	case "enable", "disable":
		if *passwordStdin {
			return fmt.Errorf("--password-stdin is not valid for %s", action)
		}
		if err := store.SetEnabled(*username, action == "enable"); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "user %s %sd\n", *username, action)
	case "list":
		if *username != "" || *passwordStdin {
			return fmt.Errorf("--username and --password-stdin are not valid for list")
		}
		users, err := store.List()
		if err != nil {
			return err
		}
		for _, user := range users {
			state := "disabled"
			if user.Enabled {
				state = "enabled"
			}
			_, _ = fmt.Fprintf(stdout, "%s\t%s\n", user.Username, state)
		}
	default:
		return fmt.Errorf("unknown user action %q", action)
	}
	return nil
}

func readPassword(reader io.Reader, enabled bool) (string, error) {
	if !enabled {
		return "", fmt.Errorf("--password-stdin is required")
	}
	password, err := bufio.NewReader(io.LimitReader(reader, 2049)).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read password: %w", err)
	}
	password = strings.TrimSuffix(password, "\n")
	password = strings.TrimSuffix(password, "\r")
	if password == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	return password, nil
}
