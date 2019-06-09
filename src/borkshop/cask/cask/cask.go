package main

import (
	"borkshop/cask"
	"borkshop/cask/caskblob"
	"borkshop/cask/caskdir"
	"borkshop/cask/caskdiskstore"
	"borkshop/cask/caskmemstore"
	"borkshop/cask/casknet"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.uber.org/multierr"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

var usage = `Content Address Store of 1KB Blocks
cask init [DIR]
  Creates a .cask directory.
  Other commands find the .cask directory in the first parent dir.
cask store [HOST:PORT] < FILE > HASH
  Stores input to CASK.
  Writes the hash.
cask load [HOST:PORT] HASH > FILE
  Writes out the file for the given hash.
cask checkin [HOST:PORT] DIR > HASH
  Stores the given directory in CASK.
  Writes the hash.
cask checkout [HOST:PORT] DIR HASH
  Writes out the directory tree from CASK to the given path.
cask serve [HOST:PORT]
  Runs a CASK server.
  Commands sent with the server's address will use the server's .cask
  instead of the local .cask.
`

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

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) (err error) {
	if len(args) < 1 {
		fmt.Fprintf(stdout, usage)
		return
	}
	command := args[0]

	// Parse and validate arguments.
	hashArg := ""
	pathArg := "."
	hostArg := ""
	peerArg := ""
	switch command {
	case "help", "-h", "--help":
		fmt.Fprintf(stdout, usage)
		return
	case "init":
		switch len(args) {
		case 1:
		case 2:
			pathArg = args[1]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[2:])
			return
		}
	case "store":
		switch len(args) {
		case 1:
		case 2:
			hostArg = "0:0"
			peerArg = args[1]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[1:])
			return
		}
	case "load":
		switch len(args) {
		case 2:
			hashArg = args[1]
		case 3:
			hostArg = "0:0"
			peerArg = args[1]
			hashArg = args[2]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[2:])
			return
		}
	case "checkin":
		switch len(args) {
		case 2:
			pathArg = args[1]
		case 3:
			hostArg = "0:0"
			peerArg = args[1]
			pathArg = args[2]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[2:])
			return
		}
	case "checkout":
		switch len(args) {
		case 3:
			pathArg = args[1]
			hashArg = args[2]
		case 4:
			hostArg = "0:0"
			peerArg = args[1]
			pathArg = args[2]
			hashArg = args[3]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[3:])
			return
		}
	case "serve":
		switch len(args) {
		case 1:
			hostArg = "0:1024"
		case 2:
			hostArg = args[1]
		default:
			err = fmt.Errorf("unexpected extra arguments: %v", args[2:])
			return
		}
	}

	// Input and dependencies.

	var hash cask.Hash
	switch command {
	case "load", "checkout":
		if h, decodeErr := hex.DecodeString(hashArg); decodeErr != nil {
			err = decodeErr
			return
		} else if len(h) != len(hash) {
			err = errors.New("invalid hash")
			return
		} else {
			copy(hash[:], h)
		}
	}

	var path string
	switch command {
	case "checkin", "checkout":
		if p, absErr := filepath.Abs(pathArg); absErr != nil {
			err = absErr
			return
		} else {
			path = p
		}
	}

	fs := osfs.New("/")

	var store cask.Store
	switch command {
	case "store", "load", "checkin", "checkout", "serve":
		if peerArg != "" {
			store = caskmemstore.New()
		} else {
			if caskPath, findErr := findCask(fs); findErr != nil {
				err = findErr
				return
			} else {
				fs := osfs.New(caskPath)
				store = &caskdiskstore.Store{Filesystem: fs}
			}
		}
	}

	var server *casknet.Server
	if hostArg != "" {
		server = &casknet.Server{
			Addr:  hostArg,
			Store: store,
		}
		if startErr := server.Start(ctx); startErr != nil {
			err = startErr
			return
		}
		defer func() {
			if stopErr := server.Stop(ctx); err != nil {
				err = multierr.Append(err, stopErr)
			}
		}()
	}

	if peerArg != "" {
		if udpAddr, resolveErr := net.ResolveUDPAddr("udp", peerArg); resolveErr != nil {
			err = resolveErr
		} else {
			store = server.Peer(udpAddr)
		}
	}

	// Execute.
	switch command {
	case "init":
		if caskPath, absErr := filepath.Abs(filepath.Join(pathArg, ".cask")); absErr != nil {
			err = absErr
			return
		} else if mkdirErr := fs.MkdirAll(caskPath, 755); mkdirErr != nil {
			err = mkdirErr
			return
		}
	case "store":
		if h, storeErr := caskblob.Store(ctx, store, os.Stdin); storeErr != nil {
			err = storeErr
			return
		} else {
			hash = h
		}
	case "load":
		if loadErr := caskblob.Load(ctx, store, os.Stdout, hash); loadErr != nil {
			err = loadErr
			return
		}
	case "checkin":
		if h, storeErr := caskdir.Store(ctx, store, fs, path); storeErr != nil {
			err = storeErr
			return
		} else {
			hash = h
		}
	case "checkout":
		if loadErr := caskdir.Load(ctx, store, fs, path, hash); loadErr != nil {
			err = loadErr
			return
		}
	case "serve":
		fmt.Fprintf(stderr, "Serving on %s\n", server.LocalAddr().String())
		<-ctx.Done()
		err = ctx.Err()
	}

	// Report.
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
