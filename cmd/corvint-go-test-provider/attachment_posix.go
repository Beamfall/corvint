//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func safeAttachmentFile(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Nlink) == 1 && uint32(stat.Uid) == uint32(os.Geteuid())
}

func supportedProviderHost() bool { return supportedProviderPlatform(runtime.GOOS, runtime.GOARCH) }

func supportedProviderPlatform(goos, goarch string) bool {
	return goos == "darwin" && goarch == "arm64"
}

func providerContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func readParentCapability() ([]byte, bool) {
	file := os.NewFile(3, "corvint-parent-capability")
	if file == nil {
		return nil, false
	}
	defer file.Close()
	value, err := io.ReadAll(io.LimitReader(file, 33))
	return value, err == nil && len(value) == 32
}

func runAuthorityCommand(ctx context.Context, executable string, argv, environment []string, cwd string, timeout time.Duration, capability []byte) (directCommandResult, error) {
	if len(capability) != 32 {
		return directCommandResult{ExitCode: -1}, errDirectCommandInvalid
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return directCommandResult{ExitCode: -1}, errDirectCommandPipe
	}
	if _, err := writer.Write(capability); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return directCommandResult{ExitCode: -1}, errDirectCommandPipe
	}
	if err := writer.Close(); err != nil {
		_ = reader.Close()
		return directCommandResult{ExitCode: -1}, errDirectCommandPipe
	}
	result, runErr := runDirectCommandWithFiles(ctx, executable, argv, environment, cwd, timeout, []*os.File{reader})
	closeErr := reader.Close()
	return result, errors.Join(runErr, closeErr)
}
