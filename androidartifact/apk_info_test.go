package androidartifact

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

func Test_GetAPKInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("setup: failed to create temp dir, error: %s", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Warnf("failed to remove temp dir, error: %s", err)
		}
	}()

	gitCommand, err := git.New(tmpDir)
	if err != nil {
		t.Fatalf("setup: failed to create git project, error: %s", err)
	}
	if err := gitCommand.Clone("https://github.com/bitrise-io/sample-artifacts.git").Run(); err != nil {
		t.Fatalf("setup: failed to clone test artifact repo, error: %s", err)
	}

	apkPath := path.Join(tmpDir, "apks", "app-debug.apk")
	got, err := GetAPKInfo(apkPath)
	if err != nil {
		t.Fatalf("GetAPKInfo() error = %v", err)
	}

	want := ApkInfo{
		AppName:           "My Application",
		PackageName:       "com.example.birmachera.myapplication",
		VersionCode:       "1",
		VersionName:       "1.0",
		MinSDKVersion:     "17",
		RawPackageContent: testArtifactAndroidManifest,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAPKInfo() = %+v, want %+v", got, want)
	}
}

const testArtifactAndroidManifest string = `<manifest xmlns:android="http://schemas.android.com/apk/res/android" android:versionCode="1" android:versionName="1.0" package="com.example.birmachera.myapplication">
	<uses-sdk android:minSdkVersion="17" android:targetSdkVersion="28"></uses-sdk>
	<uses-permission android:name="android.permission.INTERNET"></uses-permission>
	<application android:theme="null" android:label="My Application" android:icon="res/mipmap-xxxhdpi-v4/ic_launcher.png" android:debuggable="true" android:allowBackup="true" android:supportsRtl="true" android:roundIcon="res/mipmap-xxxhdpi-v4/ic_launcher_round.png" android:appComponentFactory="android.support.v4.app.CoreComponentFactory">
		<activity android:name="com.example.birmachera.myapplication.MainActivity">
			<intent-filter>
				<action android:name="android.intent.action.MAIN"></action>
				<category android:name="android.intent.category.LAUNCHER"></category>
			</intent-filter>
		</activity>
	</application>
</manifest>`
