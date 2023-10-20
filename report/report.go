package report

import (
	"fmt"
	"sync"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/api"
)

// TestReportUploader ...
type TestReportUploader struct {
	client      api.ClientAPI
	logger      log.Logger
	reportDir   string
	concurrency int
}

// NewTestReportUploader ...
func NewTestReportUploader(reportDir, buildURL, authToken string, concurrency int, logger log.Logger) TestReportUploader {
	client := api.NewBitriseClient(buildURL, authToken, logger)

	return TestReportUploader{
		client:      client,
		logger:      logger,
		reportDir:   reportDir,
		concurrency: concurrency,
	}
}

// DeployReports ...
func (t *TestReportUploader) DeployReports() []error {
	reports, err := collectReports(t.reportDir)
	if err != nil {
		return []error{err}
	}

	t.logger.Printf("Found reports (%d):", len(reports))
	for _, report := range reports {
		t.logger.Printf("- %s", report.Name)
	}

	var errors []error
	for _, report := range reports {
		if err := t.uploadReport(report); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func (t *TestReportUploader) uploadReport(report Report) error {
	t.logger.Println()
	t.logger.Printf("Uploading %s", report.Name)

	serverReport, err := t.createReport(report)
	if err != nil {
		return err
	}

	allAssetsUploaded := true
	errors := t.uploadAssets(report.Assets, serverReport.AssetURLs)
	if 0 < len(errors) {
		for _, uploadError := range errors {
			t.logger.Warnf("Asset upload failed:\n")
			t.logger.Warnf("- %w", uploadError)
		}

		allAssetsUploaded = false

		t.logger.Warnf("Html report will be marked unsuccessful as some assets could not be saved")
	}

	err = t.finishReport(serverReport.Identifier, allAssetsUploaded)
	if err != nil {
		return err
	}

	return nil
}

func (t *TestReportUploader) createReport(report Report) (ServerReport, error) {
	var assets []api.CreateReportAsset
	for _, asset := range report.Assets {
		assets = append(assets, api.CreateReportAsset{
			RelativePath: asset.TestDirRelativePath,
			FileSize:     asset.FileSize,
			ContentType:  asset.ContentType,
		})
	}

	resp, err := t.client.CreateReport(api.CreateReportParameters{
		Title:  report.Name,
		Assets: assets,
	})
	if err != nil {
		return ServerReport{}, err
	}

	urls := make(map[string]string)
	for _, assetURL := range resp.AssetURLs {
		urls[assetURL.RelativePath] = assetURL.URL
	}

	return ServerReport{
		Identifier: resp.Identifier,
		AssetURLs:  urls,
	}, nil
}

func (t *TestReportUploader) uploadAssets(assets []Asset, urls map[string]string) []error {
	var errors []error
	var wg sync.WaitGroup

	jobs := make(chan bool, t.concurrency)

	for _, item := range assets {
		wg.Add(1)

		go func(asset Asset) {
			defer wg.Done()
			defer func() {
				<-jobs
			}()

			jobs <- true

			t.logger.Debugf("Uploading %s", asset.TestDirRelativePath)

			url, ok := urls[asset.TestDirRelativePath]
			if !ok {
				errors = append(errors, fmt.Errorf("missing upload url for %s", asset.TestDirRelativePath))
				return
			}

			err := t.client.UploadAsset(url, asset.Path, asset.ContentType)
			if err != nil {
				errors = append(errors, err)
			}
		}(item)
	}

	wg.Wait()

	return errors
}

func (t *TestReportUploader) finishReport(identifier string, allAssetsUploaded bool) error {
	return t.client.FinishReport(identifier, allAssetsUploaded)
}
