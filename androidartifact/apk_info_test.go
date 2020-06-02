package androidartifact

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

func TestParsePackageInfos(t *testing.T) {

	tests := []struct {
		name    string
		aaptOut string
		want    string
		want1   string
		want2   string
	}{
		{
			name:    "test_with_empty_platformBuildVersionName",
			aaptOut: `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0' platformBuildVersionName=''`,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_without_platformBuildVersionName",
			aaptOut: `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0'`,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_with_platformBuildVersionName",
			aaptOut: `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0' platformBuildVersionName='3'`,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_without_name",
			aaptOut: `package: name='' versionCode='1' versionName='1.0' platformBuildVersionName='3'`,
			want:    "",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_without_name_and_versionCode",
			aaptOut: `package: name='' versionCode='' versionName='1.0' platformBuildVersionName='3'`,
			want:    "",
			want1:   "",
			want2:   "1.0",
		},
		{
			name:    "test_without_name_and_versionCode_and_versionName",
			aaptOut: `package: name='' versionCode='' versionName='' platformBuildVersionName='3'`,
			want:    "",
			want1:   "",
			want2:   "",
		},
		{
			name:    "test_without_name_and_versionCode_and_versionName",
			aaptOut: `package: name='' versionCode='2' versionName='' platformBuildVersionName='3'`,
			want:    "",
			want1:   "2",
			want2:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := ParsePackageInfos(tt.aaptOut)
			if got != tt.want {
				t.Errorf("ParsePackageInfos() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParsePackageInfos() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("ParsePackageInfos() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_parseAPKInfo(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
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

	tests := []struct {
		name    string
		apkPath string
		want    ApkInfo
		wantErr bool
	}{
		{
			name:    "",
			apkPath: path.Join(tmpDir, "apks", "app-debug.apk"),
			want: ApkInfo{
				AppName:           "My Application",
				PackageName:       "com.example.birmachera.myapplication",
				VersionCode:       "1",
				VersionName:       "1.0",
				MinSDKVersion:     "17",
				RawPackageContent: testArtifactAndroidManifest,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAPKInfo(tt.apkPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseAPKInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAPKInfo() = %+v, want %+v", got, tt.want)
			}

			gotFallback, err := getAPKInfoWithAapt(tt.apkPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("getAPKInfoWithAapt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check all fields, exluding RawPackageContent, which is used for debugging only
			if gotFallback.AppName != tt.want.AppName {
				t.Errorf("getAPKInfoWithAapt().AppName = %+v, want %+v", gotFallback.AppName, tt.want)
			}
			if gotFallback.PackageName != tt.want.PackageName {
				t.Errorf("getAPKInfoWithAapt().PackageName = %+v, want %+v", gotFallback.PackageName, tt.want)
			}
			if gotFallback.VersionCode != tt.want.VersionCode {
				t.Errorf("getAPKInfoWithAapt().VersionCode = %+v, want %+v", gotFallback.VersionCode, tt.want)
			}
			if gotFallback.VersionName != tt.want.VersionName {
				t.Errorf("getAPKInfoWithAapt().VersionName = %+v, want %+v", gotFallback.VersionName, tt.want)
			}
			if gotFallback.MinSDKVersion != tt.want.MinSDKVersion {
				t.Errorf("getAPKInfoWithAapt().MinSDKVersion = %+v, want %+v", gotFallback.MinSDKVersion, tt.want)
			}
		})
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
