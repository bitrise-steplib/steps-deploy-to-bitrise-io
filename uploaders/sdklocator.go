package uploaders

import (
	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

// sdkLocator resolves the Android SDK only when a build tool is asked for, which happens on the aapt
// fallback of APK parsing. Stacks without an Android SDK still deploy their artifacts.
type sdkLocator struct {
	envRepository env.Repository
}

func NewSDKLocator(envRepository env.Repository) *sdkLocator {
	return &sdkLocator{envRepository: envRepository}
}

func (l *sdkLocator) LatestBuildToolPath(name string) (string, error) {
	sdkModel, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    l.envRepository.Get("ANDROID_HOME"),
		AndroidSDKRoot: l.envRepository.Get("ANDROID_SDK_ROOT"),
	}, pathutil.NewPathChecker())
	if err != nil {
		return "", err
	}
	return sdkModel.LatestBuildToolPath(name)
}
