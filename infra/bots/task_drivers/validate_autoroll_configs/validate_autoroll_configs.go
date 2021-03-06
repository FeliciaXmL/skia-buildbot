package main

/*
	Read and validate an AutoRoll config file.
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/flynn/json5"
	"github.com/pmezard/go-difflib/difflib"
	"go.skia.org/infra/autoroll/go/roller"
	"go.skia.org/infra/go/skerr"
	"go.skia.org/infra/go/sklog"
	"go.skia.org/infra/go/util"
	"go.skia.org/infra/task_driver/go/lib/os_steps"
	"go.skia.org/infra/task_driver/go/td"
)

var (
	// Required properties for this task.
	projectId = flag.String("project_id", "", "ID of the Google Cloud project.")
	taskId    = flag.String("task_id", "", "ID of this task.")
	taskName  = flag.String("task_name", "", "Name of the task.")
	workdir   = flag.String("workdir", ".", "Working directory")
	config    = flag.String("config", "", "Config file or dir of config files to validate.")

	// Optional flags.
	local  = flag.Bool("local", false, "True if running locally (as opposed to on the bots)")
	output = flag.String("o", "", "If provided, dump a JSON blob of step data to the given file. Prints to stdout if '-' is given.")
)

func validateConfig(ctx context.Context, f string) (string, error) {
	var rollerName string
	return rollerName, td.Do(ctx, td.Props(fmt.Sprintf("Validate %s", f)), func(ctx context.Context) error {
		// Decode the config.
		var cfg roller.AutoRollerConfig
		if err := util.WithReadFile(f, func(r io.Reader) error {
			// TODO(borenet): This will just ignore any extraneous keys!
			return json5.NewDecoder(r).Decode(&cfg)
		}); err != nil {
			return fmt.Errorf("Failed to read %s: %s", f, err)
		}

		// Re-encode the config back to JSON. Note that the encoder respects
		// struct field ordering, whereas it sorts map keys; in order to obtain
		// a comparable JSON encoding, we'll have to decode it again into a map,
		// then encode yet again.
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(cfg); err != nil {
			return skerr.Wrap(err)
		}
		var m map[string]interface{}
		if err := json.NewDecoder(&buf).Decode(&m); err != nil {
			return skerr.Wrap(err)
		}
		expectedBytes, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return skerr.Wrap(err)
		}
		expected := string(expectedBytes)

		// Read the original config file into a flat map[string]interface{},
		// then re-encode it as JSON to strip the comments.
		m = nil
		if err := util.WithReadFile(f, func(r io.Reader) error {
			return json5.NewDecoder(r).Decode(&m)
		}); err != nil {
			return skerr.Wrap(err)
		}
		actualBytes, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return skerr.Wrap(err)
		}
		actual := string(actualBytes)
		if actual != expected {
			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(expected),
				B:        difflib.SplitLines(actual),
				FromFile: "Expected",
				ToFile:   "Actual",
				Context:  3,
				Eol:      "\n",
			})
			if err != nil {
				return skerr.Wrap(err)
			}
			return skerr.Fmt("Config file %q contains unused keys:\n%s", f, diff)
		}

		// Validate the config. We do this after the above checks, because
		// Validate() may propagate some shared configuration entries downward,
		// and that would cause the above comparison to fail.
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("%s failed validation: %s", f, err)
		}

		rollerName = cfg.RollerName
		return nil
	})
}

func main() {
	// Setup.
	ctx := td.StartRun(projectId, taskId, taskName, output, local)
	defer td.EndRun(ctx)

	if *config == "" {
		td.Fatalf(ctx, "--config is required.")
	}

	// Gather files to validate.
	configFiles := []string{}
	f, err := os_steps.Stat(ctx, *config)
	if err != nil {
		td.Fatal(ctx, err)
	}
	if f.IsDir() {
		files, err := os_steps.ReadDir(ctx, *config)
		if err != nil {
			td.Fatal(ctx, err)
		}
		for _, f := range files {
			// Ignore subdirectories and file names starting with '.'
			if !f.IsDir() && f.Name()[0] != '.' {
				configFiles = append(configFiles, filepath.Join(*config, f.Name()))
			}
		}
	} else {
		configFiles = append(configFiles, *config)
	}

	// Validate the file(s).
	rollers := make(map[string]string, len(configFiles))
	for _, f := range configFiles {
		rollerName, err := validateConfig(ctx, f)
		if err != nil {
			td.Fatal(ctx, err)
		}
		if otherFile, ok := rollers[rollerName]; ok {
			td.Fatalf(ctx, "Roller %q is defined in both %s and %s", rollerName, f, otherFile)
		}
		rollers[rollerName] = f
	}
	sklog.Infof("Validated %d files.", len(configFiles))
}
