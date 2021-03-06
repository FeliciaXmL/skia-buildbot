package main

import (
	"context"

	"github.com/spf13/cobra"

	"go.skia.org/infra/gold-client/go/goldclient"
	"go.skia.org/infra/golden/go/types"
)

// diffEnv provides the environment for the diff command.
type diffEnv struct {
	inputFile  string
	test       string
	corpus     string
	instanceID string
	outDir     string
	workDir    string
}

// getDiffCmd returns the definition of the diff command.
func getDiffCmd() *cobra.Command {
	env := &diffEnv{}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compute the diff to the closest known image",
		Long: `
Downloads all images for a given test and compares them to the provided image.

Outputs the closest of these images and the diff to the given folder.
`,
		Run: env.runDiffCmd,
	}

	cmd.Flags().StringVar(&env.inputFile, "input", "", "path to file to diff")
	cmd.Flags().StringVar(&env.test, "test", "", "Test name")
	cmd.Flags().StringVar(&env.corpus, "corpus", "", "Corpus name")
	cmd.Flags().StringVar(&env.instanceID, "instance", "", "Instance (e.g. 'chrome', 'flutter')")
	cmd.Flags().StringVar(&env.outDir, "out-dir", "", "Work directory that will contain the output")
	cmd.Flags().StringVar(&env.workDir, fstrWorkDir, "", "Work directory for intermediate results")
	// Everything is required for this command.
	Must(cmd.MarkFlagRequired(fstrWorkDir))
	Must(cmd.MarkFlagRequired("input"))
	Must(cmd.MarkFlagRequired("test"))
	Must(cmd.MarkFlagRequired("corpus"))
	Must(cmd.MarkFlagRequired("out-dir"))
	Must(cmd.MarkFlagRequired("instance"))

	return cmd
}

// runDiffCmd executes the diff logic for comparing a given image against all that Gold knows.
func (d *diffEnv) runDiffCmd(cmd *cobra.Command, args []string) {
	auth, err := goldclient.LoadAuthOpt(d.workDir)
	ifErrLogExit(cmd, err)

	if auth == nil {
		logErrf(cmd, "Auth is empty - did you call goldctl auth first?")
		exitProcess(cmd, 1)
	}

	config := goldclient.GoldClientConfig{
		InstanceID: d.instanceID,
		WorkDir:    d.workDir,
	}

	// overwrite any existing configs in this workdir.
	goldClient, err := goldclient.NewCloudClient(auth, config)
	ifErrLogExit(cmd, err)

	err = goldClient.Diff(context.Background(), types.TestName(d.test), d.corpus, d.inputFile, d.outDir)
	ifErrLogExit(cmd, err)
}
