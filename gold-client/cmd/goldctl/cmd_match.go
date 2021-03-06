package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"go.skia.org/infra/go/skerr"
	"go.skia.org/infra/go/util"
	"go.skia.org/infra/gold-client/go/imgmatching"
	"go.skia.org/infra/gold-client/go/imgmatching/fuzzy"
	"go.skia.org/infra/gold-client/go/imgmatching/sobel"
)

// matchEnv provides the environment for the match command.
type matchEnv struct {
	algorithmName string
	parameters    []string
}

// getMatchCmd returns the definition of the match command.
func getMatchCmd() *cobra.Command {
	env := &matchEnv{}
	cmd := &cobra.Command{
		Use:   "match",
		Short: "Runs an image matching algorithm against two images on disk",
		Long: `
Takes a (potentially non-exact) image matching algorithm (and any algorithm-specific parameters)
and executes it against the two given images on disk.

Reports whether or not the specified algorithm considers the two images to be equivalent.

This command is intended for experimenting with various combinations of non-exact image matching
algorithms and parameters.
`,
		Args: cobra.ExactArgs(2), // Takes exactly two images as positional arguments.
		Run:  env.runMatchCmd,
	}

	cmd.Flags().StringVar(&env.algorithmName, "algorithm", "", "Image matching algorithm (e.g. exact, fuzzy, sobel).")
	cmd.Flags().StringArrayVar(&env.parameters, "parameter", []string{}, "Any number of algorithm-specific parameters represented as name:value pairs (e.g. sobel_edge_threshold:10).")
	Must(cmd.MarkFlagRequired("algorithm"))

	return cmd
}

// runMatchCmd instantiates the specified image matching algorithm and runs it against two images.
func (m *matchEnv) runMatchCmd(cmd *cobra.Command, args []string) {
	// Load input images.
	leftImage, err := loadPng(args[0])
	ifErrLogExit(cmd, err)
	rightImage, err := loadPng(args[1])
	ifErrLogExit(cmd, err)

	// Instantiate the specified algorithm.
	optionalKeys, err := makeOptionalKeys(m.algorithmName, m.parameters)
	ifErrLogExit(cmd, err)
	algorithmName, matcher, err := imgmatching.MakeMatcher(optionalKeys)
	ifErrLogExit(cmd, err)

	// Run the algorithm against the two input images.
	imagesMatch := matcher.Match(leftImage, rightImage)

	// Report the algorithm's boolean output.
	if imagesMatch {
		fmt.Println("Images match.")
	} else {
		fmt.Println("Images do not match.")
	}

	// Print out algorithm-specific debug information.
	switch algorithmName {
	case imgmatching.FuzzyMatching:
		printOutFuzzyDebugInfo(matcher.(*fuzzy.Matcher))
	case imgmatching.SobelFuzzyMatching:
		err := printOutSobelDebugInfo(matcher.(*sobel.Matcher))
		ifErrLogExit(cmd, err)
	}

	exitProcess(cmd, 0)
}

// loadPng loads a PNG image from disk.
func loadPng(fileName string) (image.Image, error) {
	// Load the image and save the bytes because we need to return them.
	imgBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, skerr.Wrapf(err, "loading file %s", fileName)
	}
	img, err := png.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, skerr.Wrapf(err, "decoding PNG file %s", fileName)
	}
	return img, nil
}

// makeOptionalKeys turns the specified algorithm name and parameters into the equivalent map of
// optional keys that function imgmatching.MakeMatcher() expects.
func makeOptionalKeys(algorithmName string, parameters []string) (map[string]string, error) {
	keys := map[string]string{
		imgmatching.AlgorithmNameOptKey: algorithmName,
	}

	for _, pair := range parameters {
		split := strings.SplitN(pair, ":", 2)
		if len(split) != 2 {
			return nil, skerr.Fmt("parameter %q must be a key:value pair", pair)
		}
		keys[split[0]] = split[1]
	}

	return keys, nil
}

// printOutFuzzyDebugInfo prints out stats reported by the given fuzzy.Matcher.
func printOutFuzzyDebugInfo(matcher *fuzzy.Matcher) {
	printDebugInfoItem("Number of different pixels", matcher.NumDifferentPixels())
	printDebugInfoItem("Maximum per-channel delta sum", matcher.MaxPixelDelta())
}

// printOutSobelDebugInfo writes intermediate images generated by the given sobel.Matcher to a
// temporary directory and prints out the resulting paths. It also prints out the stats reported by
// the embedded fuzzy.Matcher.
func printOutSobelDebugInfo(matcher *sobel.Matcher) error {
	tempDir, err := ioutil.TempDir("", "goldctl-*")
	if err != nil {
		return skerr.Wrap(err)
	}

	writeToDiskAndPrintOut := func(msg, filename string, img image.Image) error {
		path := path.Join(tempDir, filename)
		err := writePngToTmp(img, path)
		if err != nil {
			return skerr.Wrap(err)
		}
		printDebugInfoItem(msg, path)
		return nil
	}

	err = writeToDiskAndPrintOut("Sobel operator output (left image)", "sobel-output.png", matcher.SobelOutput())
	if err != nil {
		return skerr.Wrap(err)
	}

	err = writeToDiskAndPrintOut("Left image with edges removed", "image1-no-edges.png", matcher.ExpectedImageWithEdgesRemoved())
	if err != nil {
		return skerr.Wrap(err)
	}

	err = writeToDiskAndPrintOut("Right image with edges removed", "image2-no-edges.png", matcher.ActualImageWithEdgesRemoved())
	if err != nil {
		return skerr.Wrap(err)
	}

	printOutFuzzyDebugInfo(&matcher.Matcher)
	return nil
}

// writePngToTmp takes an image, saves it to disk as a PNG image with the given filename.
func writePngToTmp(img image.Image, filename string) error {
	err := util.WithWriteFile(filename, func(writer io.Writer) error {
		err := png.Encode(writer, img)
		if err != nil {
			return skerr.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return skerr.Wrap(err)
	}
	return nil
}

// printDebugInfoItem prints out the given debug name/value pair. The name is left-padded so that
// multiple calls to this function produce vertically aligned output that is easier to read.
func printDebugInfoItem(name string, value interface{}) {
	fmt.Printf("%36s: %v\n", name, value)
}
