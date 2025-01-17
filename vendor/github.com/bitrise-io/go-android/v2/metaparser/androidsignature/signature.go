package androidsignature

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-utils/command"
)

const (
	unsignedJarSignatureMessage = "jar is unsigned"
	validJarSignatureMessage    = "jar verified"
	validV2PlusSignatureMessage = "Verifies"
)

var (
	NotVerifiedError      = errors.New("not verified")
	NoSignatureFoundError = errors.New("no signature found")
)

// Read ...
//
// Deprecated: Read is deprecated. Use ReadAABSignature or ReadAPKSignature method instead.
func Read(path string) (string, error) {
	return ReadAABSignature(path)
}

// ReadAABSignature returns the signature of the provided AAB file.
// If the signature can't be read (unsigned, unexpected certificate printing format, ...), it returns a NoSignatureFoundError.
// If the signature is not verified, it returns a NotVerifiedError.
func ReadAABSignature(path string) (string, error) {
	return getJarSignature(path)
}

// ReadAPKSignature returns the signature of the provided APK file.
// If the signature can't be read (unsigned, unexpected certificate printing format, ...), it returns a NoSignatureFoundError.
// If the signature is not verified, it returns a NotVerifiedError.
func ReadAPKSignature(apkPath string) (string, error) {
	idSigPath := apkPath + ".idsig"
	if _, err := os.Stat(idSigPath); err == nil {
		signature, err := getV4Signature(apkPath, idSigPath)
		if err != nil && !errors.Is(err, NotVerifiedError) && !errors.Is(err, NoSignatureFoundError) {
			return "", err
		}
		if signature != "" {
			return signature, nil
		}
	}

	signature, err := getV23Signature(apkPath)
	if err != nil && !errors.Is(err, NotVerifiedError) && !errors.Is(err, NoSignatureFoundError) {
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
	})
	if err != nil {
		return "", fmt.Errorf("failed to create sdk model, error: %s", err)
	}

	apkSignerPath, err := sdkModel.LatestBuildToolPath("apksigner")
	if err != nil {
		return "", fmt.Errorf("failed to find latest aapt binary, error: %s", err)
	}

	params := append([]string{"verify", "--print-certs", "-v"}, pathParams...)
	apkSignerOutput, err := command.New(apkSignerPath, params...).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if strings.Contains(apkSignerOutput, `DOES NOT VERIFY`) {
				return "", NotVerifiedError
			}
		}
		return "", err
	}

	if !strings.Contains(apkSignerOutput, validV2PlusSignatureMessage) {
		return "", NotVerifiedError
	}

	// The signature details appear in the output in the following format:
	// Signer #1 certificate DN: C=Aa, ST=Bbbbb, L=Ccccc, O=Ddddd, OU=Eeeee, CN=Fffff
	// Signer #1 certificate SHA-256 digest: <hash>
	// Signer #1 certificate SHA-1 digest: <hash>
	// Signer #1 certificate MD5 digest: <hash>
	regex := regexp.MustCompile("Signer #1 certificate DN: (.*)")
	res := regex.FindAllStringSubmatch(apkSignerOutput, 1)
	if len(res) > 0 && len(res[0]) > 1 {
		return res[0][1], nil
	}

	return "", NoSignatureFoundError
}

func getJarSignature(path string) (string, error) {
	params := []string{"-verify", "-certs", "-verbose", path}
	output, err := command.New("jarsigner", params...).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", err
	}

	if strings.Contains(output, unsignedJarSignatureMessage) {
		return "", NoSignatureFoundError
	}

	if !strings.Contains(output, validJarSignatureMessage) {
		return "", NotVerifiedError
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

	return "", NoSignatureFoundError
}
