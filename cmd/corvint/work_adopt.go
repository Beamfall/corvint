package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Beamfall/corvint/internal/worklistadapter"
	"github.com/Beamfall/corvint/internal/workqueue"
)

// workAdapterPath is the committed adapter `work init` writes. The observer runs it
// under the fixed VPO-V0-022 PATH, so it names `corvint` bare (WQO-V0-047).
const workAdapterPath = ".corvint/work-queue-adapter"

const workAdapterScript = `#!/bin/bash
# repository-work-queue-adapter/0 written by ` + "`corvint work init`" + `. Corvint runs it
# with a fixed PATH; install corvint in /opt/homebrew/bin or /usr/local/bin.
exec corvint work adapter "$@"
`

const workEmptyWorklist = `{"profile":"corvint-worklist/0","tickets":[]}
`

// workAdoptionVerbs are the two work operations outside work-command-result/0:
// they print their own output and never reach the observer.
var workAdoptionVerbs = map[string]func(context.Context, string, []string, io.Writer, io.Writer) int{
	"adapter": runWorkAdapter,
	"init":    runWorkInit,
}

func runWorkAdoption(ctx context.Context, root string, arguments []string, stdout, stderr io.Writer) (int, bool) {
	if len(arguments) == 0 {
		return 0, false
	}
	verb, found := workAdoptionVerbs[arguments[0]]
	if !found {
		return 0, false
	}
	return verb(ctx, root, arguments[1:], stdout, stderr), true
}

// runWorkAdapter is the repository-work-queue-adapter/0 producer for a committed
// worklist (WQO-V0-048). It reads only qualified source and writes one document.
func runWorkAdapter(ctx context.Context, root string, arguments []string, stdout, stderr io.Writer) int {
	if err := worklistadapter.Run(ctx, "corvint work adapter", root, arguments, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

// runWorkInit writes the three adoption files and refuses to replace any (WQO-V0-047).
func runWorkInit(_ context.Context, root string, arguments []string, stdout, stderr io.Writer) int {
	name, err := parseWorkInitArguments(arguments)
	if err != nil {
		fmt.Fprintln(stderr, "corvint work init:", err)
		return 2
	}
	policy, err := workAdoptionPolicy(name)
	if err != nil {
		fmt.Fprintln(stderr, "corvint work init:", err)
		return 2
	}
	worklistPath, _ := worklistadapter.WorklistPath(worklistadapter.RepositoryMapping)
	files := []workAdoptionFile{
		{worklistadapter.PolicyPath, policy.Canonical(), 0644},
		{worklistPath, []byte(workEmptyWorklist), 0644},
		{workAdapterPath, []byte(workAdapterScript), 0755},
	}
	for _, file := range files {
		if _, err := os.Lstat(filepath.Join(root, file.path)); !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "corvint work init: %s already exists; nothing was written\n", file.path)
			return 2
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".corvint"), 0755); err != nil {
		fmt.Fprintln(stderr, "corvint work init:", err)
		return 2
	}
	for _, file := range files {
		if err := file.write(root); err != nil {
			fmt.Fprintln(stderr, "corvint work init:", err)
			return 2
		}
		fmt.Fprintln(stdout, file.path)
	}
	return 0
}

func parseWorkInitArguments(arguments []string) (string, error) {
	if len(arguments) == 2 && arguments[0] == "--repository" {
		return arguments[1], nil
	}
	if len(arguments) == 1 && len(arguments[0]) > len("--repository=") && arguments[0][:len("--repository=")] == "--repository=" {
		return arguments[0][len("--repository="):], nil
	}
	return "", errors.New("usage: corvint [--root PATH] work init --repository NAME")
}

// workAdoptionPolicy derives every authority from one owner-chosen repository name
// and returns the policy only after the closed WQO-V0-001 parser accepts it.
func workAdoptionPolicy(name string) (*workqueue.Policy, error) {
	operation := func(verb string) []string { return []string{verb} }
	policy := &workqueue.Policy{
		AccessContextID: "access:" + name + ":local", AdapterPath: workAdapterPath,
		AdapterProfile: "repository-work-queue-adapter/0", DetailLimit: "512",
		MappingVersion: worklistadapter.RepositoryMapping,
		Operations:     workqueue.PolicyOperations{Details: operation("details"), Snapshot: operation("snapshot"), Verify: operation("verify")},
		Profile:        workqueue.PolicyProfile, QueueAuthorityID: "queue:" + name + ":worklist",
		RepositoryAuthorityID: "repo:" + name, ScopeID: "scope:" + name + ":worklist",
	}
	policy.RefreshIdentity()
	parsed, err := workqueue.ParsePolicy(policy.Canonical())
	if err != nil {
		return nil, fmt.Errorf("repository name %q is not a valid authority token", name)
	}
	return parsed, nil
}

type workAdoptionFile struct {
	path string
	raw  []byte
	mode os.FileMode
}

func (file workAdoptionFile) write(root string) error {
	handle, err := os.OpenFile(filepath.Join(root, file.path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, file.mode)
	if err != nil {
		return err
	}
	if _, err = handle.Write(file.raw); err != nil {
		handle.Close()
		return err
	}
	if err = handle.Chmod(file.mode); err != nil {
		handle.Close()
		return err
	}
	return handle.Close()
}
