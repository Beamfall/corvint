//go:build darwin || linux

package main

import (
	"os"
	"syscall"
)

func producerReadFlags() int { return os.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_NONBLOCK }
