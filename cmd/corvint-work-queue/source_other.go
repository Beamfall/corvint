//go:build !darwin && !linux

package main

import "os"

func producerReadFlags() int { return os.O_RDONLY }
