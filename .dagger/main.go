// A generated module for Venom functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/venom/internal/dagger"
	"fmt"
	"golang.org/x/mod/modfile"
	"path/filepath"
	"runtime"
	"strconv"
)

type Venom struct{}

func (m *Venom) Build(ctx context.Context, source *dagger.Directory) (*dagger.Container, error) {
	buildEnv, err := m.BuildEnv(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("error creating build environment: %w", err)
	}

	workDir, err := buildEnv.Workdir(ctx)
	if err != nil {
		return nil, fmt.Errorf("error determining working directory of build container: %w", err)
	}

	build := buildEnv.
		WithExec([]string{"make", "build", "OS=" + runtime.GOOS, "ARCH=" + runtime.GOARCH}).
		Directory(filepath.Join(workDir, "dist"))

	return dag.Container().From("alpine:3.16").
		WithFile("/usr/local/venom", build.File(fmt.Sprintf("venom.%s-%s", runtime.GOOS, runtime.GOARCH))), nil
}

type TestResult struct {
	ResultsDir *dagger.Directory
	ExitCode   int
}

func (m *Venom) Test(ctx context.Context, source *dagger.Directory, tests *dagger.Directory, results *dagger.Directory) (TestResult, error) {
	venom, err := m.Build(ctx, source)
	if err != nil {
		return TestResult{}, fmt.Errorf("error building venom: %w", err)
	}

	const workDir = "/workdir"
	var (
		testsDir   = filepath.Join(workDir, "tests")
		resultsDir = filepath.Join(workDir, "results")
	)

	testContainer, err := venom.WithWorkdir(workDir).
		WithMountedDirectory(testsDir, tests).
		WithMountedDirectory(resultsDir, results).
		WithEnvVariable("VENOM_OUTPUT_DIR", resultsDir).
		WithEnvVariable("VENOM_LIB_DIR", filepath.Join(testsDir, "lib")).
		WithEnvVariable("VENOM_VERBOSE", "1").
		WithExec([]string{"/bin/sh", "-c", "/usr/local/venom run ./tests/*.y*ml; echo -n $? > exit_code"}).
		Sync(ctx)
	if err != nil {
		return TestResult{}, fmt.Errorf("unexpected error executing tests: %w", err)
	}

	testResult := TestResult{ResultsDir: testContainer.Directory(resultsDir)}

	exitCode, err := testContainer.File(filepath.Join(workDir, "exit_code")).Contents(ctx)
	if err != nil {
		return testResult, fmt.Errorf("could not get error code from test command: %w", err)
	}

	exitCodeInt, err := strconv.Atoi(exitCode)
	if err != nil {
		return testResult, fmt.Errorf("invalid exit code for tests: %w", err)
	}
	testResult.ExitCode = exitCodeInt

	return testResult, nil
}

func (m *Venom) BuildEnv(ctx context.Context, source *dagger.Directory) (*dagger.Container, error) {
	modContents, err := source.File("go.mod").Contents(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading go.mod: %w", err)
	}

	modFile, err := modfile.ParseLax("go.mod", []byte(modContents), nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing go.mod: %w", err)
	}

	return dag.Container().
		From("golang:"+modFile.Go.Version).
		WithWorkdir("/usr/src/app").
		WithFiles("/usr/src/app", []*dagger.File{source.File("go.mod"), source.File("go.sum")}).
		WithExec([]string{"/bin/sh", "-c", "go mod download && go mod verify"}).
		WithDirectory("/usr/src/app", source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{".dagger", "tests", "results"},
		}), nil
}
