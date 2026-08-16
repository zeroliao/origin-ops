package authn

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const (
	credentialFileVersion = 1
	passwordIterations    = 600_000
	passwordSaltBytes     = 16
	passwordHashBytes     = 32
	maxCredentialFileSize = 1 << 20
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,63}$`)

type Principal struct {
	Username string
	Revision uint64
}

type User struct {
	Username string
	Enabled  bool
	Revision uint64
}

type Store struct {
	path string
}

type credentialFile struct {
	Version int          `json:"version"`
	Users   []userRecord `json:"users"`
}

type userRecord struct {
	Username   string `json:"username"`
	Enabled    bool   `json:"enabled"`
	Revision   uint64 `json:"revision"`
	Algorithm  string `json:"algorithm"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Hash       string `json:"hash"`
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Add(username, password string) error {
	username, err := validateCredentials(username, password)
	if err != nil {
		return err
	}
	file, err := s.read()
	if err != nil {
		return err
	}
	for _, user := range file.Users {
		if strings.EqualFold(user.Username, username) {
			return fmt.Errorf("user %q already exists", username)
		}
	}
	record, err := newUserRecord(username, password, 1, true)
	if err != nil {
		return err
	}
	file.Users = append(file.Users, record)
	return s.write(file)
}

func (s *Store) SetPassword(username, password string) error {
	username, err := validateCredentials(username, password)
	if err != nil {
		return err
	}
	file, err := s.read()
	if err != nil {
		return err
	}
	index := findUser(file.Users, username)
	if index < 0 {
		return fmt.Errorf("user %q does not exist", username)
	}
	revision := file.Users[index].Revision + 1
	file.Users[index], err = newUserRecord(file.Users[index].Username, password, revision, file.Users[index].Enabled)
	if err != nil {
		return err
	}
	return s.write(file)
}

func (s *Store) SetEnabled(username string, enabled bool) error {
	username, err := validateUsername(username)
	if err != nil {
		return err
	}
	file, err := s.read()
	if err != nil {
		return err
	}
	index := findUser(file.Users, username)
	if index < 0 {
		return fmt.Errorf("user %q does not exist", username)
	}
	if file.Users[index].Enabled == enabled {
		return nil
	}
	file.Users[index].Enabled = enabled
	file.Users[index].Revision++
	return s.write(file)
}

func (s *Store) List() ([]User, error) {
	file, err := s.read()
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(file.Users))
	for _, record := range file.Users {
		users = append(users, User{Username: record.Username, Enabled: record.Enabled, Revision: record.Revision})
	}
	sort.Slice(users, func(i, j int) bool { return strings.ToLower(users[i].Username) < strings.ToLower(users[j].Username) })
	return users, nil
}

func (s *Store) Authenticate(username, password string) (Principal, bool, error) {
	file, err := s.read()
	if err != nil {
		return Principal{}, false, err
	}
	index := findUser(file.Users, username)
	if index < 0 {
		_, _ = pbkdf2.Key(sha256.New, password, []byte("origin-ops-dummy"), passwordIterations, passwordHashBytes)
		return Principal{}, false, nil
	}
	record := file.Users[index]
	valid, err := verifyPassword(record, password)
	if err != nil {
		return Principal{}, false, err
	}
	if !valid || !record.Enabled {
		return Principal{}, false, nil
	}
	return Principal{Username: record.Username, Revision: record.Revision}, true, nil
}

func (s *Store) Validate(principal Principal) (bool, error) {
	file, err := s.read()
	if err != nil {
		return false, err
	}
	index := findUser(file.Users, principal.Username)
	if index < 0 {
		return false, nil
	}
	record := file.Users[index]
	return record.Enabled && record.Revision == principal.Revision, nil
}

func (s *Store) CheckPermissions() error {
	info, err := os.Stat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat credential file: %w", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("credential file permissions must not allow group or other access")
	}
	return nil
}

func (s *Store) read() (credentialFile, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return credentialFile{Version: credentialFileVersion, Users: []userRecord{}}, nil
	}
	if err != nil {
		return credentialFile{}, fmt.Errorf("open credential file: %w", err)
	}
	defer file.Close()
	var payload credentialFile
	decoder := json.NewDecoder(io.LimitReader(file, maxCredentialFileSize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return credentialFile{}, fmt.Errorf("decode credential file: %w", err)
	}
	if payload.Version != credentialFileVersion {
		return credentialFile{}, fmt.Errorf("unsupported credential file version %d", payload.Version)
	}
	if err := ensureCredentialEOF(decoder); err != nil {
		return credentialFile{}, err
	}
	for _, record := range payload.Users {
		if _, err := validateUsername(record.Username); err != nil {
			return credentialFile{}, fmt.Errorf("invalid credential user: %w", err)
		}
		if record.Revision == 0 || record.Algorithm != "pbkdf2-sha256" || record.Iterations < 100_000 {
			return credentialFile{}, fmt.Errorf("invalid password metadata for user %q", record.Username)
		}
	}
	return payload, nil
}

func (s *Store) write(payload credentialFile) error {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential file: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".users-*.tmp")
	if err != nil {
		return fmt.Errorf("create credential file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure credential file: %w", err)
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return fmt.Errorf("write credential file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync credential file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close credential file: %w", err)
	}
	if err := replaceFile(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace credential file: %w", err)
	}
	return nil
}

func replaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	}
	backup, err := os.CreateTemp(filepath.Dir(destination), ".users-*.bak")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	backup.Close()
	os.Remove(backupPath)
	if err := os.Rename(destination, backupPath); err != nil {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		_ = os.Rename(backupPath, destination)
		return err
	}
	return os.Remove(backupPath)
}

func newUserRecord(username, password string, revision uint64, enabled bool) (userRecord, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return userRecord{}, fmt.Errorf("generate password salt: %w", err)
	}
	hash, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordHashBytes)
	if err != nil {
		return userRecord{}, fmt.Errorf("derive password hash: %w", err)
	}
	return userRecord{
		Username: username, Enabled: enabled, Revision: revision,
		Algorithm: "pbkdf2-sha256", Iterations: passwordIterations,
		Salt: base64.RawStdEncoding.EncodeToString(salt), Hash: base64.RawStdEncoding.EncodeToString(hash),
	}, nil
}

func verifyPassword(record userRecord, password string) (bool, error) {
	salt, err := base64.RawStdEncoding.DecodeString(record.Salt)
	if err != nil {
		return false, fmt.Errorf("decode password salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(record.Hash)
	if err != nil {
		return false, fmt.Errorf("decode password hash: %w", err)
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, record.Iterations, len(expected))
	if err != nil {
		return false, fmt.Errorf("derive password hash: %w", err)
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func validateCredentials(username, password string) (string, error) {
	username, err := validateUsername(username)
	if err != nil {
		return "", err
	}
	if len(password) < 8 || len(password) > 1024 {
		return "", fmt.Errorf("password must contain between 8 and 1024 bytes")
	}
	return username, nil
}

func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return "", fmt.Errorf("username must contain 3-64 letters, numbers, dots, underscores, or hyphens")
	}
	return username, nil
}

func findUser(users []userRecord, username string) int {
	for index, user := range users {
		if strings.EqualFold(user.Username, strings.TrimSpace(username)) {
			return index
		}
	}
	return -1
}

func ensureCredentialEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode credential file: multiple JSON values")
		}
		return fmt.Errorf("decode credential file: %w", err)
	}
	return nil
}
