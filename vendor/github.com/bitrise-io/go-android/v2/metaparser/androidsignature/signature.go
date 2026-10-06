package androidsignature

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/hashicorp/go-version"
)

var cmdFactory = command.NewFactory(env.NewRepository())

const (
	unsignedJarSignatureMessage = "jar is unsigned"
	validJarSignatureMessage    = "jar verified"
	validV2PlusSignatureMessage = "Verifies"
)

var (
	ErrNotVerified      = errors.New("not verified")
	ErrNoSignatureFound = errors.New("no signature found")
)

// Read ...
//
// Deprecated: Read is deprecated. Use ReadAABSignature or ReadAPKSignature method instead.
func Read(path string) (string, error) {
	return ReadAABSignature(path)
}

// ReadAABSignature returns the signature of the provided AAB file.
// If the signature can't be read (unsigned, unexpected certificate printing format, ...), it returns a ErrNoSignatureFound.
// If the signature is not verified, it returns a ErrNotVerified.
func ReadAABSignature(path string) (string, error) {
	return getJarSignature(path)
}

// ReadAPKSignature returns the signature of the provided APK file.
// If the signature can't be read (unsigned, unexpected certificate printing format, ...), it returns a ErrNoSignatureFound.
// If the signature is not verified, it returns a ErrNotVerified.
func ReadAPKSignature(apkPath string) (string, error) {
	idSigPath := apkPath + ".idsig"
	if _, err := os.Stat(idSigPath); err == nil {
		signature, err := getV4Signature(apkPath, idSigPath)
		if err != nil && !errors.Is(err, ErrNotVerified) && !errors.Is(err, ErrNoSignatureFound) {
			return "", err
		}
		if signature != "" {
			return signature, nil
		}
	}

	signature, err := getV23Signature(apkPath)
	if err != nil && !errors.Is(err, ErrNotVerified) && !errors.Is(err, ErrNoSignatureFound) {
		return "", err
	}
	if signature != "" {
		return signature, nil
	}

	return getJarSignature(apkPath)
}

func getV4Signature(apkPath string, idsigPath string) (string, error) {
	if _, err := os.Stat(idsigPath); err != nil {
		return "", fmt.Errorf("failed to check if detached signature file (.idsig) exist: %s", err)
	}

	pathParams := []string{"-v4-signature-file", idsigPath, apkPath}
	return getV2PlusSignature(pathParams)
}

func getV23Signature(path string) (string, error) {
	pathParams := []string{path}
	return getV2PlusSignature(pathParams)
}

func getV2PlusSignature(pathParams []string) (string, error) {
	sdkModel, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    os.Getenv("ANDROID_HOME"),
		AndroidSDKRoot: os.Getenv("ANDROID_SDK_ROOT"),
	}, pathutil.NewPathChecker())
	if err != nil {
		return "", fmt.Errorf("failed to create sdk model, error: %s", err)
	}

	apkSignerPath, err := sdkModel.LatestBuildToolPath("apksigner")
	if err != nil {
		return "", fmt.Errorf("failed to find latest aapt binary, error: %s", err)
	}

	params := append([]string{"verify", "--print-certs", "-v"}, pathParams...)
	apkSignerOutput, err := cmdFactory.Create(apkSignerPath, params, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if strings.Contains(apkSignerOutput, `DOES NOT VERIFY`) {
				return "", ErrNotVerified
			}
		}
		return "", err
	}

	if !strings.Contains(apkSignerOutput, validV2PlusSignatureMessage) {
		return "", ErrNotVerified
	}

	// apksigner (Android SDK Build-Tools) changed its output format starting with build-tools 37:
	// - build-tools < 37: "Signer #1 certificate DN: ..."
	// - build-tools >= 37: "V2 Signer: certificate DN: ..." (scheme label is V1/V2/V3.0/V3.1/V3.2/V4)
	regex := regexp.MustCompile(`Signer #1 certificate DN: (.*)`)
	if apkSignerVersion, err := version.NewVersion(filepath.Base(filepath.Dir(apkSignerPath))); err == nil && apkSignerVersion.GreaterThanOrEqual(version.Must(version.NewVersion("37.0.0"))) {
		regex = regexp.MustCompile(`V\d+(?:\.\d+)? Signer: certificate DN: (.*)`)
	}

	res := regex.FindAllStringSubmatch(apkSignerOutput, 1)
	if len(res) > 0 && len(res[0]) > 1 {
		return res[0][1], nil
	}

	return "", ErrNoSignatureFound
}

func getJarSignature(path string) (string, error) {
	params := []string{"-verify", "-certs", "-verbose", path}
	output, err := cmdFactory.Create("jarsigner", params, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", err
	}

	if strings.Contains(output, unsignedJarSignatureMessage) {
		return "", ErrNoSignatureFound
	}

	if !strings.Contains(output, validJarSignatureMessage) {
		return "", ErrNotVerified
	}

	var signature string

	// The signature details appear in the output in the following format:
	// - Signed by "C=Aa, ST=Bbbbb, L=Ccccc, O=Ddddd, OU=Eeeee, CN=Fffff"
	regex := regexp.MustCompile(`- Signed by ".*"`)
	sig := regex.FindString(output)
	if sig != "" {
		signature = strings.TrimPrefix(sig, "- Signed by \"")
		signature = strings.TrimSuffix(signature, "\"")
		return signature, nil
	}

	return "", ErrNoSignatureFound
}
