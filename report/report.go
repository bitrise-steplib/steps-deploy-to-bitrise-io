package report

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/api"
)

// HTMLReportUploader ...
type HTMLReportUploader struct {
	client      api.ClientAPI
	logger      log.Logger
	reportDir   string
	concurrency int
}

// NewHTMLReportUploader ...
func NewHTMLReportUploader(reportDir, buildURL, authToken string, concurrency int, logger log.Logger) HTMLReportUploader {
	client := api.NewBitriseClient(buildURL, authToken, logger)

	return HTMLReportUploader{
		client:      client,
		logger:      logger,
		reportDir:   reportDir,
		concurrency: concurrency,
	}
}

// DeployReports ...
func (h *HTMLReportUploader) DeployReports() []error {
	reports, err := collectReports(h.reportDir)
	if err != nil {
		return []error{err}
	}

	h.logger.Printf("Found reports (%d):", len(reports))
	for _, report := range reports {
		h.logger.Printf("- %s", report.Name)
	}

	validatedReports, validationErrors := h.validate(reports)
	if len(validationErrors) != 0 {
		h.logger.Warnf("Validation errors:\n")

		for _, validationError := range validationErrors {
			h.logger.Warnf("- %s\n", validationError)
		}
	}

	var uploadErrors []error
	for _, report := range validatedReports {
		if err := h.uploadReport(report); err != nil {
			uploadErrors = append(uploadErrors, err)
		}
	}

	return uploadErrors
}

func (h *HTMLReportUploader) validate(reports []Report) ([]Report, []error) {
	var validatedReports []Report
	var validationErrors []error

	for _, report := range reports {
		valid := false

		for _, asset := range report.Assets {
			if filepath.Base(asset.Path) == "index.html" {
				valid = true
				break
			}
		}

		if valid {
			validatedReports = append(validatedReports, report)
			continue
		}

		validationErrors = append(validationErrors, fmt.Errorf("missing index.html file for %s", report.Name))
	}

	return validatedReports, validationErrors
}

func (h *HTMLReportUploader) uploadReport(report Report) error {
	h.logger.Println()
	h.logger.Printf("Uploading %s", report.Name)

	serverReport, err := h.createReport(report)
	if err != nil {
		return err
	}

	allAssetsUploaded := true
	errors := h.uploadAssets(report.Assets, serverReport.AssetURLs)
	if 0 < len(errors) {
		for _, uploadError := range errors {
			h.logger.Warnf("Asset upload failed:\n")
			h.logger.Warnf("- %w", uploadError)
		}

		allAssetsUploaded = false

		h.logger.Warnf("Html report will be marked unsuccessful as some assets could not be saved")
	}

	err = h.finishReport(serverReport.Identifier, allAssetsUploaded)
	if err != nil {
		return err
	}

	return nil
}

func (h *HTMLReportUploader) createReport(report Report) (ServerReport, error) {
	var assets []api.CreateReportAsset
	for _, asset := range report.Assets {
		assets = append(assets, api.CreateReportAsset{
			RelativePath: asset.TestDirRelativePath,
			FileSize:     asset.FileSize,
			ContentType:  asset.ContentType,
		})
	}

	resp, err := h.client.CreateReport(api.CreateReportParameters{
		Title:    report.Name,
		Category: report.Info.Category,
		Assets:   assets,
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

func (h *HTMLReportUploader) uploadAssets(assets []Asset, urls map[string]string) []error {
	var errors []error
	var wg sync.WaitGroup

	jobs := make(chan bool, h.concurrency)

	for _, item := range assets {
		wg.Add(1)

		go func(asset Asset) {
			defer wg.Done()
			defer func() {
				<-jobs
			}()

			jobs <- true

			h.logger.Debugf("Uploading %s", asset.TestDirRelativePath)

			url, ok := urls[asset.TestDirRelativePath]
			if !ok {
				errors = append(errors, fmt.Errorf("missing upload url for %s", asset.TestDirRelativePath))
				return
			}

			err := h.client.UploadAsset(url, asset.Path, asset.ContentType)
			if err != nil {
				errors = append(errors, err)
			}
		}(item)
	}

	wg.Wait()

	return errors
}

func (h *HTMLReportUploader) finishReport(identifier string, allAssetsUploaded bool) error {
	return h.client.FinishReport(identifier, allAssetsUploaded)
}
