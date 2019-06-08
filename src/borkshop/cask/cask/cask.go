package main

import (
	"borkshop/cask"
	"borkshop/cask/caskblob"
	"borkshop/cask/caskdir"
	"borkshop/cask/caskdiskstore"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		signal.Reset()
		cancel()
	}()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("command name is a required argument")
	}
	command := args[0]

	var hashArg string
	var pathArg string
	switch command {
	case "init":
		switch len(args) {
		case 1:
			pathArg = "."
		case 2:
			pathArg = args[1]
		default:
			return fmt.Errorf("unexpected extra arguments: %v", args[2:])
		}
	case "store":
		if len(args) != 1 {
			return fmt.Errorf("unexpected extra arguments: %v", args[1:])
		}
	case "load":
		if len(args) != 2 {
			return fmt.Errorf("unexpected extra arguments: %v", args[2:])
		}
		hashArg = args[1]
	case "checkin":
		if len(args) != 2 {
			return fmt.Errorf("unexpected extra arguments: %v", args[2:])
		}
		pathArg = args[1]
	case "checkout":
		if len(args) != 3 {
			return fmt.Errorf("unexpected extra arguments: %v", args[3:])
		}
		pathArg = args[1]
		hashArg = args[2]
	}

	var hash cask.Hash
	switch command {
	case "load", "checkout":
		if h, err := hex.DecodeString(hashArg); err != nil {
			return err
		} else if len(h) != len(hash) {
			return errors.New("invalid hash")
		} else {
			copy(hash[:], h)
		}
	}

	var path string
	switch command {
	case "checkin", "checkout":
		if p, err := filepath.Abs(pathArg); err != nil {
			return err
		} else {
			path = p
		}
	}

	fs := osfs.New("/")

	var store cask.Store
	switch command {
	case "store", "load", "checkin", "checkout":
		if caskPath, err := findCask(fs); err != nil {
			return err
		} else {
			fs := osfs.New(caskPath)
			store = &caskdiskstore.Store{Filesystem: fs}
		}
	}

	switch command {
	case "init":
		if caskPath, err := filepath.Abs(filepath.Join(pathArg, ".cask")); err != nil {
			return err
		} else if err := fs.MkdirAll(caskPath, 755); err != nil {
			return err
		}
	case "store":
		if h, err := caskblob.Store(ctx, store, os.Stdin); err != nil {
			return err
		} else {
			hash = h
		}
	case "load":
		if err := caskblob.Load(ctx, store, os.Stdout, hash); err != nil {
			return err
		}
	case "checkin":
		if h, err := caskdir.Store(ctx, store, fs, path); err != nil {
			return err
		} else {
			hash = h
		}
	case "checkout":
		if err := caskdir.Load(ctx, store, fs, path, hash); err != nil {
			return err
		}
	}

	switch command {
	case "store", "checkin":
		fmt.Fprintf(stdout, "%x\n", hash)
	}

	return nil
}

func findCask(fs billy.Filesystem) (string, error) {
	cwd, err := filepath.Abs("")
	if err != nil {
		return "", err
	}
	at := cwd
	for {
		caskPath := filepath.Join(at, ".cask")
		if s, err := fs.Stat(caskPath); err == nil && s.IsDir() {
			return caskPath, nil
		}

		parent := filepath.Join(at, "..")
		if parent == at {
			return "", fmt.Errorf("use cask init [<path>] to create a CAS1KB")
		}
		at = parent
	}
}
