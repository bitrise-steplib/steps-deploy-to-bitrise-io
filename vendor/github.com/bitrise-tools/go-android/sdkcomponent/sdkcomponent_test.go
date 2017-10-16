package sdkcomponent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemImageComponent(t *testing.T) {
	{
		component := SystemImage{
			Platform: "android-24",
			Tag:      "",
			ABI:      "x86",
		}

		require.Equal(t, "system-images;android-24;default;x86", component.GetSDKStylePath())
		require.Equal(t, "sys-img-x86-android-24", component.GetLegacySDKStylePath())
		require.Equal(t, "system-images/android-24/default/x86", component.InstallPathInAndroidHome())
	}

	{
		component := SystemImage{
			Platform: "android-24",
			Tag:      "default",
			ABI:      "x86",
		}

		require.Equal(t, "system-images;android-24;default;x86", component.GetSDKStylePath())
		require.Equal(t, "sys-img-x86-android-24", component.GetLegacySDKStylePath())
		require.Equal(t, "system-images/android-24/default/x86", component.InstallPathInAndroidHome())
	}

	{
		component := SystemImage{
			Platform: "android-23",
			Tag:      "google_apis",
			ABI:      "armeabi-v7a",
		}

		require.Equal(t, "system-images;android-23;google_apis;armeabi-v7a", component.GetSDKStylePath())
		require.Equal(t, "sys-img-armeabi-v7a-google_apis-23", component.GetLegacySDKStylePath())
		require.Equal(t, "system-images/android-23/google_apis/armeabi-v7a", component.InstallPathInAndroidHome())
	}

	{
		component := SystemImage{
			Platform: "android-23",
			Tag:      "android-tv",
			ABI:      "armeabi-v7a",
		}

		require.Equal(t, "system-images;android-23;android-tv;armeabi-v7a", component.GetSDKStylePath())
		require.Equal(t, "sys-img-armeabi-v7a-android-tv-23", component.GetLegacySDKStylePath())
		require.Equal(t, "system-images/android-23/android-tv/armeabi-v7a", component.InstallPathInAndroidHome())
	}
}

func TestPlatformComponent(t *testing.T) {
	component := Platform{
		Version: "android-23",
	}

	require.Equal(t, "platforms;android-23", component.GetSDKStylePath())
	require.Equal(t, "android-23", component.GetLegacySDKStylePath())
	require.Equal(t, "platforms/android-23", component.InstallPathInAndroidHome())
}

func TestBuildToolComponent(t *testing.T) {
	component := BuildTool{
		Version: "19.1.0",
	}

	require.Equal(t, "build-tools;19.1.0", component.GetSDKStylePath())
	require.Equal(t, "build-tools-19.1.0", component.GetLegacySDKStylePath())
	require.Equal(t, "build-tools/19.1.0", component.InstallPathInAndroidHome())
}
