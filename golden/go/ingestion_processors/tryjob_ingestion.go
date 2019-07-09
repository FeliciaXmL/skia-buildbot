package ingestion_processors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.skia.org/infra/go/auth"
	"go.skia.org/infra/go/ds"
	"go.skia.org/infra/go/eventbus"
	"go.skia.org/infra/go/gerrit"
	"go.skia.org/infra/go/httputils"
	"go.skia.org/infra/go/human"
	"go.skia.org/infra/go/ingestion"
	"go.skia.org/infra/go/paramtools"
	"go.skia.org/infra/go/sharedconfig"
	"go.skia.org/infra/go/skerr"
	"go.skia.org/infra/go/sklog"
	"go.skia.org/infra/go/util"
	"go.skia.org/infra/go/vcsinfo"
	"go.skia.org/infra/golden/go/bbstate"
	"go.skia.org/infra/golden/go/config"
	"go.skia.org/infra/golden/go/tryjobstore"
	"go.skia.org/infra/golden/go/tryjobstore/ds_tryjobstore"
	"go.skia.org/infra/golden/go/types"
	gstorage "google.golang.org/api/storage/v1"
)

// Define configuration options to be used in the config file under
// ExtraParams.
const (
	CONFIG_GERRIT_CODE_REVIEW_URL     = "GerritCodeReviewURL"
	CONFIG_SERVICE_ACCOUNT_FILE       = "ServiceAccountFile"
	CONFIG_BUILD_BUCKET_URL           = "BuildBucketURL"
	CONFIG_BUILD_BUCKET_NAME          = "BuildBucketName"
	CONFIG_BUILD_BUCKET_POLL_INTERVAL = "BuildBucketPollInterval"
	CONFIG_BUILD_BUCKET_TIME_WINDOW   = "BuildBucketTimeWindow"
	CONFIG_BUILDER_REGEX              = "BuilderRegEx"
	CONFIG_JOB_CFG_FILE               = "JobConfigFile"
)

// Register the ingestion Processor with the ingestion framework.
func init() {
	ingestion.Register(config.CONSTRUCTOR_GOLD_TRYJOB, newGoldTryjobProcessor)
}

// goldTryjobProcessor implements the ingestion.Processor interface to ingest
// tryjob results.
type goldTryjobProcessor struct {
	buildIssueSync bbstate.BuildIssueSync
	tryjobStore    tryjobstore.TryjobStore
	vcs            vcsinfo.VCS
	cfgFile        string
	syncMonitor    *util.CondMonitor
}

// newGoldTryjobProcessor implements the ingestion.Constructor function.
func newGoldTryjobProcessor(vcs vcsinfo.VCS, config *sharedconfig.IngesterConfig, ignoreClient *http.Client, eventBus eventbus.EventBus) (ingestion.Processor, error) {
	gerritURL := config.ExtraParams[CONFIG_GERRIT_CODE_REVIEW_URL]
	if strings.TrimSpace(gerritURL) == "" {
		return nil, fmt.Errorf("Missing URL for the Gerrit code review systems. Got value: '%s'", gerritURL)
	}

	// Get the config options.
	svcAccountFile := config.ExtraParams[CONFIG_SERVICE_ACCOUNT_FILE]
	sklog.Infof("Got service account file '%s'", svcAccountFile)

	pollInterval, err := parseDuration(config.ExtraParams[CONFIG_BUILD_BUCKET_POLL_INTERVAL], bbstate.DefaultPollInterval)
	if err != nil {
		return nil, err
	}

	timeWindow, err := parseDuration(config.ExtraParams[CONFIG_BUILD_BUCKET_TIME_WINDOW], bbstate.DefaultTimeWindow)
	if err != nil {
		return nil, err
	}

	buildBucketURL := config.ExtraParams[CONFIG_BUILD_BUCKET_URL]
	buildBucketName := config.ExtraParams[CONFIG_BUILD_BUCKET_NAME]
	if (buildBucketURL == "") || (buildBucketName == "") {
		return nil, fmt.Errorf("BuildBucketName and BuildBucketURL must not be empty.")
	}

	builderRegExp := config.ExtraParams[CONFIG_BUILDER_REGEX]
	if builderRegExp == "" {
		builderRegExp = bbstate.DefaultTestBuilderRegex
	}

	// Get the config file in the repo that should be parsed to determine whether a
	// bot uploads results. Currently only applies to the Skia repo.
	cfgFile := config.ExtraParams[CONFIG_JOB_CFG_FILE]

	// Create the cloud tryjob store.
	tryjobStore, err := ds_tryjobstore.New(ds.DS, eventBus)
	if err != nil {
		return nil, fmt.Errorf("Error creating tryjob store: %s", err)
	}
	sklog.Infof("Cloud Tryjobstore Store created")

	// Instantiate the Gerrit API client.
	ts, err := auth.NewJWTServiceAccountTokenSource("", svcAccountFile, gstorage.CloudPlatformScope, "https://www.googleapis.com/auth/userinfo.email")
	if err != nil {
		return nil, fmt.Errorf("Failed to authenticate service account: %s", err)
	}
	client := httputils.DefaultClientConfig().WithTokenSource(ts).With2xxOnly().Client()
	sklog.Infof("HTTP client instantiated")

	gerritReview, err := gerrit.NewGerrit(gerritURL, "", client)
	if err != nil {
		return nil, err
	}
	sklog.Infof("Gerrit client instantiated")

	bbConf := &bbstate.Config{
		BuildBucketURL:  buildBucketURL,
		BuildBucketName: buildBucketName,
		Client:          client,
		TryjobStore:     tryjobStore,
		GerritClient:    gerritReview,
		PollInterval:    pollInterval,
		TimeWindow:      timeWindow,
		BuilderRegexp:   builderRegExp,
	}

	bbGerritClient, err := bbstate.NewBuildBucketState(bbConf)
	if err != nil {
		return nil, err
	}
	sklog.Infof("BuildBucketState created")

	ret := &goldTryjobProcessor{
		buildIssueSync: bbGerritClient,
		tryjobStore:    tryjobStore,
		vcs:            vcs,
		cfgFile:        cfgFile,

		// The argument to NewCondMonitor is 1 because we always want exactly one go-routine per unique
		// issue ID to enter the critical section. See the syncIssueAndTryjob function.
		syncMonitor: util.NewCondMonitor(1),
	}
	eventBus.SubscribeAsync(tryjobstore.EV_TRYJOB_UPDATED, ret.tryjobUpdatedHandler)

	return ret, nil
}

// See ingestion.Processor interface.
func (g *goldTryjobProcessor) Process(ctx context.Context, resultsFile ingestion.ResultFileLocation) error {
	dmResults, err := processDMResults(resultsFile)
	if err != nil {
		sklog.Errorf("Error processing result: %s", err)
		return ingestion.IgnoreResultsFileErr
	}

	// Make sure we have an issue, patchset and a buildbucket id.
	if (dmResults.Issue <= 0) || (dmResults.Patchset <= 0) || (dmResults.BuildBucketID <= 0) {
		sklog.Errorf("Invalid data. issue, patchset and buildbucket id must be > 0. Got (%d, %d, %d).", dmResults.Issue, dmResults.Patchset, dmResults.BuildBucketID)
		return ingestion.IgnoreResultsFileErr
	}

	entries, err := extractTraceDBEntries(dmResults)
	if err != nil {
		sklog.Errorf("Error getting tracedb entries: %s", err)
		return ingestion.IgnoreResultsFileErr
	}

	// Save the results to the trybot store.
	issueID := dmResults.Issue
	tryjob, err := g.tryjobStore.GetTryjob(issueID, dmResults.BuildBucketID)
	if err != nil {
		sklog.Errorf("Error retrieving tryjob: %s", err)
		return ingestion.IgnoreResultsFileErr
	}

	tryjob, err = g.syncIssueAndTryjob(issueID, tryjob, dmResults, resultsFile.Name())
	if err != nil {
		return err
	}

	// Add the Githash of the underlying result.
	if tryjob.MasterCommit == "" {
		tryjob.MasterCommit = dmResults.GitHash
	} else if tryjob.MasterCommit != dmResults.GitHash {
		sklog.Errorf("Master commit in tryjob and ingested results do not match.")
		return ingestion.IgnoreResultsFileErr
	}

	// Convert to a trybotstore.TryjobResult slice by aggregating parameters for each test/digest pair.
	resultsMap := make(map[string]*tryjobstore.TryjobResult, len(entries))
	for _, entry := range entries {
		testName := types.TestName(entry.Params[types.PRIMARY_KEY_FIELD])
		key := string(testName) + string(entry.Value)
		if found, ok := resultsMap[key]; ok {
			found.Params.AddParams(entry.Params)
		} else {
			resultsMap[key] = &tryjobstore.TryjobResult{
				BuildBucketID: tryjob.BuildBucketID,
				TestName:      testName,
				Digest:        types.Digest(entry.Value),
				Params:        paramtools.NewParamSet(entry.Params),
			}
		}
	}

	tjResults := make([]*tryjobstore.TryjobResult, 0, len(resultsMap))
	for _, result := range resultsMap {
		tjResults = append(tjResults, result)
	}

	// Update the database with the results.
	if err := g.tryjobStore.UpdateTryjobResult(tjResults); err != nil {
		return skerr.Fmt("Error updating tryjob results: %s", err)
	}

	tryjob.Status = tryjobstore.TRYJOB_INGESTED
	if err := g.tryjobStore.UpdateTryjob(0, tryjob, nil); err != nil {
		return err
	}
	sklog.Infof("Ingested: %s", resultsFile.Name())
	return nil
}

func (g *goldTryjobProcessor) syncIssueAndTryjob(issueID int64, tryjob *tryjobstore.Tryjob, dmResults *dmResults, resultFileName string) (*tryjobstore.Tryjob, error) {
	// Only let one thread in at time for each issueID. In most cases they will follow the fast
	// path of finding the issue and having an non-nil tryjob.
	defer g.syncMonitor.Enter(issueID).Release()

	// Fetch the issue and check if the trybot is contained.
	issue, err := g.tryjobStore.GetIssue(issueID, false)
	if err != nil {
		return nil, skerr.Fmt("Unable to retrieve issue %d to process file %s. Got error: %s", issueID, resultFileName, err)
	}

	// If we haven't loaded the tryjob then see if we can fetch it from
	// Gerrit and Buildbucket. This should be the exception since tryjobs should
	// be picket up by BuildBucketState as they are added.
	if (tryjob == nil) || (issue == nil) || !issue.HasPatchset(tryjob.PatchsetID) {
		var err error
		if issue, tryjob, err = g.buildIssueSync.SyncIssueTryjob(issueID, dmResults.BuildBucketID); err != nil {
			sklog.Errorf("Error fetching the issue and tryjob information: %s", err)
			return nil, ingestion.IgnoreResultsFileErr
		}

		// If the issue is nil, that means it could not be found (404) and we don't want to ingest this
		// file again (and avoid repeated errors).
		if issue == nil {
			sklog.Errorf("Unable to find issue %d. Received 404 from Gerrit.", issueID)
			return nil, ingestion.IgnoreResultsFileErr
		}
	}
	return tryjob, nil
}

// setTryjobToState is a utility function that updates the status of a tryjob
// to the new status if the new status is a logical successor to the current status.
// If tryjob is nil, issueID and tryjobID will be used to fetch the tryjob record.
func (g *goldTryjobProcessor) setTryjobToStatus(tryjob *tryjobstore.Tryjob, minStatus tryjobstore.TryjobStatus, newStatus tryjobstore.TryjobStatus) error {
	return g.tryjobStore.UpdateTryjob(tryjob.BuildBucketID, nil, func(curr interface{}) interface{} {
		tryjob := curr.(*tryjobstore.Tryjob)
		// Only update if it is at the minimum status level and the status increases.
		if (tryjob.Status < minStatus) || (tryjob.Status >= newStatus) {
			return nil
		}

		tryjob.Status = newStatus
		return tryjob
	})
}

// See ingestion.Processor interface.
func (g *goldTryjobProcessor) BatchFinished() error { return nil }

// parseDuration parses the given duration. If strVal is empty the default value
// will be returned. If the given duration is invalid an error will be returned.
func parseDuration(strVal string, defaultVal time.Duration) (time.Duration, error) {
	if strVal == "" {
		return defaultVal, nil
	}
	return human.ParseDuration(strVal)
}

// tryjobUpdateHandler is the event handler for when a tryjob is updated in the
// underlying tryjob store.
func (g *goldTryjobProcessor) tryjobUpdatedHandler(evData interface{}) {
	// Make a shallow copy of the tryjob.
	tryjob := *(evData.(*tryjobstore.Tryjob))

	// Check if this is a no-upload bot. If that's the case mark the bot as ingested.
	if g.noUpload(tryjob.Builder, tryjob.MasterCommit) {
		// Mark as ingested if it has completed.
		if err := g.setTryjobToStatus(&tryjob, tryjobstore.TRYJOB_COMPLETE, tryjobstore.TRYJOB_INGESTED); err != nil {
			sklog.Errorf("Unable to set tryjob (%d, %d) to status 'ingested': %s", tryjob.IssueID, tryjob.BuildBucketID, err)
		}
		sklog.Infof("Job %d for issue %d marked as ingested.", tryjob.BuildBucketID, tryjob.IssueID)
	}
}

// TODO(stephana): Make the noUpload code use the same code as gen_tasks.go in the
// skia repo. This is essentially a copy of the code, but uses the same source of
// information (the cfg.json file from skia).

// noUpload returns true if this builder does not upload results and we should
// therefore not wait for results to appear.
func (g *goldTryjobProcessor) noUpload(builder, commit string) bool {
	if g.cfgFile == "" {
		return false
	}

	ctx := context.Background()
	cfgContent, err := g.vcs.GetFile(ctx, g.cfgFile, commit)
	if err != nil {
		sklog.Errorf("Error retrieving %s: %s", g.cfgFile, err)
	}

	// Parse the config file used to generate the tasks.
	config := struct {
		NoUpload []string `json:"no_upload"`
	}{}
	if err := json.Unmarshal([]byte(cfgContent), &config); err != nil {
		sklog.Errorf("Unable to parse %s. Got error: %s", g.cfgFile, err)
	}

	// See if we match the builders that should not be uploaded.
	for _, s := range config.NoUpload {
		m, err := regexp.MatchString(s, builder)
		if err != nil {
			sklog.Errorf("Error matching regex: %s", err)
			continue
		}
		if m {
			return true
		}
	}
	return false
}