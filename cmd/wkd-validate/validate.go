package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"

	"github.com/spf13/pflag"
	"github.com/tv42/zbase32"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	pflag.Parse()
	ctx := logger.WithContext(context.Background())

	gpgPath, err := exec.LookPath("gpg")
	if err != nil {
		logger.Fatal().Msg("You need to have gpg installed in your PATH.")
	}

	for _, email := range pflag.Args() {
		u, err := calculateWKDURL(email)
		if err != nil {
			logger.Fatal().Err(err).Msgf("Failed to generate WKD URL for %s,", email)
		}
		if err := validateURLContent(ctx, u, gpgPath); err != nil {
			logger.Fatal().Err(err).Msg("Failed to validate WKD content.")
		}
	}
}

func validateURLContent(ctx context.Context, url string, gpgPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d while fetching key via WKD", resp.StatusCode)
	}
	tmpfile, err := ioutil.TempFile(os.TempDir(), "wkd-validate")
	if err != nil {
		return fmt.Errorf("failed to generate temporary directory: %w", err)
	}
	defer func() {
		tmpfile.Close()
		os.RemoveAll(tmpfile.Name())
	}()
	defer resp.Body.Close()
	var data bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(tmpfile, &data), resp.Body); err != nil {
		return fmt.Errorf("failed to write response to temporary directory: %w", err)
	}
	tmpfile.Close()
	if bytes.Contains(data.Bytes(), []byte("-----BEGIN PGP")) {
		return fmt.Errorf("data appears to be a key in ASCII armor")
	}
	return exec.CommandContext(ctx, gpgPath, "--with-colons", tmpfile.Name()).Run()
}

func calculateWKDURL(email string) (string, error) {
	elems := strings.Split(email, "@")
	if len(elems) != 2 {
		return "", errors.New("invalid email address")
	}
	s := sha1.New()
	if _, err := s.Write([]byte(elems[0])); err != nil {
		return "", fmt.Errorf("failed to generate sha1 of name component: %w", err)
	}
	enc := zbase32.EncodeToString(s.Sum([]byte{}))
	return fmt.Sprintf("https://%s/.well-known/openpgpkey/hu/%s", elems[1], enc), nil
}
