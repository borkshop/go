package main

import (
	"borkshop/cask"
	"borkshop/cask/blob"
	"borkshop/cask/dir"
	"borkshop/cask/diskstore"
	"borkshop/cask/memstore"
	"borkshop/cask/net"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
cask load [HOST:PORT] HASH[:PATH] > FILE
  Writes out the file for the given hash.
cask checkin [HOST:PORT] DIR > HASH
  Stores the given directory in CASK.
  Writes the hash.
cask checkout [HOST:PORT] DIR HASH[:PATH]
  Writes out the directory tree from CASK to the given path.
cask ls/list [HOST:PORT] HASH[:PATH]
  Writes the list of entries in the directory with the hash.
cask hash [HOST:PORT] HASH:PATH
  Follows a path from the hash of a directory.
  Writes the hash of the addressed object.
cask serve [HOST:PORT]
  Runs a CASK server.
  Commands sent with the server's address will use the server's .cask
  instead of the local .cask.
cask path
  Writes the location of the nearest .cask directory.
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

	fs := osfs.New("/")

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, fs); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, fs billy.Filesystem) (err error) {
	if len(args) < 1 {
		fmt.Fprintf(stdout, usage)
		return
	}

	command := args[0]

	// Parse and validate arguments.
	hashArg := ""
	pathArg := ""
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
			err = fmt.Errorf("usage error: cask %s [DIR]: 0 or 1 but got %d arguments", command, len(args)-1)
			return
		}
	case "store":
		switch len(args) {
		case 1:
		case 2:
			hostArg = "0:0"
			peerArg = args[1]
		default:
			err = fmt.Errorf("usage error: cask %s [HOST:PORT]: 0 or 1 but got %d arguments", command, len(args)-1)
			return
		}
	case "load", "checkin", "list", "ls", "hash":
		switch len(args) {
		case 2:
			hashArg = args[1]
		case 3:
			hostArg = "0:0"
			peerArg = args[1]
			hashArg = args[2]
		default:
			err = fmt.Errorf("usage error: cask %s [HOST:PORT] HASH: 1 or 2 but got %d arguments", command, len(args)-1)
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
			err = fmt.Errorf("usage error: cask %s [HOST:PORT] PATH HASH: 2 or 3 but got %d arguments", command, len(args)-1)
			return
		}
	case "serve":
		switch len(args) {
		case 1:
			hostArg = "0:1024"
		case 2:
			hostArg = args[1]
		default:
			err = fmt.Errorf("usage error: cask %s [HOST:PORT]: 0 or 1 but got %d arguments", command, len(args)-1)
			return
		}
	case "path":
		switch len(args) {
		case 1:
		default:
			err = fmt.Errorf("usage error: cask %s: 0 but got %d arguments", command, len(args)-1)
			return
		}
	default:
		err = fmt.Errorf("usage error: unrecognized command: %s", args[0])
		return
	}

	// Input and dependencies.

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

	var store cask.Store
	switch command {
	case "store", "load", "checkin", "checkout", "list", "ls", "hash", "serve":
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

	var hash cask.Hash
	switch command {
	case "load", "checkout", "list", "ls", "hash":
		if h, resolveErr := resolve(ctx, store, hashArg); resolveErr != nil {
			err = resolveErr
			return
		} else {
			hash = h
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
	case "list", "ls":
		if list, loadErr := caskdir.List(ctx, store, hash); loadErr != nil {
			err = loadErr
			return
		} else {
			for _, entry := range list {
				mode := "?"
				switch entry.Mode {
				case caskdir.FileMode:
					mode = "f"
				case caskdir.ExecMode:
					mode = "x"
				case caskdir.DirMode:
					mode = "d"
				}
				fmt.Fprintf(stdout, "%x %s %s\n", entry.Hash, mode, string(entry.Name))
			}
		}
	case "hash":
		if entry, resolveErr := caskdir.Resolve(ctx, store, hash, pathArg); resolveErr != nil {
			err = resolveErr
			return
		} else {
			hash = entry.Hash
		}
	case "serve":
		fmt.Fprintf(stderr, "Serving on %s\n", server.LocalAddr().String())
		<-ctx.Done()
		err = ctx.Err()
	case "path":
		if caskPath, findErr := findCask(fs); findErr != nil {
			err = findErr
			return
		} else {
			fmt.Fprintf(stdout, "%s\n", caskPath)
		}
	}

	// Report.
	switch command {
	case "store", "checkin", "hash":
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

func resolve(ctx context.Context, store cask.Store, hashArg string) (cask.Hash, error) {
	parts := strings.SplitN(hashArg, ":", 2)
	hashArg = parts[0]
	var path string
	if len(parts) == 2 {
		path = parts[1]
	}

	var hash cask.Hash
	if h, err := hex.DecodeString(hashArg); err != nil {
		return hash, err
	} else if len(h) != len(hash) {
		return hash, errors.New("invalid hash")
	} else {
		copy(hash[:], h)
	}

	if path != "" {
		if entry, err := caskdir.Resolve(ctx, store, hash, path); err != nil {
			return hash, err
		} else {
			hash = entry.Hash
		}
	}

	return hash, nil
}
