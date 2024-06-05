package androidartifact

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/bundletool"
	"github.com/kr/pretty"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

func Test_GetAABInfo(t *testing.T) {
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

	bt, err := bundletool.New("1.15.0")
	if err != nil {
		t.Fatalf("setup: failed to initialize bundletool, error: %s", err)
	}

	aapPath := path.Join(tmpDir, "app-bitrise-signed.aab")
	got, err := GetAABInfo(bt, aapPath)
	if err != nil {
		t.Fatalf("GetAABInfo() error = %v", err)
	}

	want := AabInfo{
		AppName:           "sample-apps-android-simple",
		PackageName:       "com.bitrise_io.sample_apps_android_simple",
		VersionCode:       "189",
		VersionName:       "1.0",
		MinSDKVersion:     "15",
		RawPackageContent: testAABArtifactAndroidManifest,
	}
	if diffs := pretty.Diff(got, want); len(diffs) > 0 {
		t.Errorf(
			"\nGetAABInfo()\n - got:\t\t%+v\n - want:\t%+v\n diff:\n\t%s",
			got,
			want,
			strings.Join(diffs, "\n"),
		)
	}
}

const testAABArtifactAndroidManifest string = `<manifest xmlns:android="http://schemas.android.com/apk/res/android" android:versionCode="189" android:versionName="1.0" package="com.bitrise_io.sample_apps_android_simple" platformBuildVersionCode="189" platformBuildVersionName="1.0">
      
  <uses-sdk android:minSdkVersion="15" android:targetSdkVersion="26"/>
      
  <application android:allowBackup="true" android:icon="@mipmap/ic_launcher" android:label="@string/app_name" android:supportsRtl="true" android:theme="@style/AppTheme">
            
    <activity android:label="@string/app_name" android:name="com.bitrise_io.sample_apps_android_simple.MainActivity">
                  
      <intent-filter>
                        
        <action android:name="android.intent.action.MAIN"/>
                        
        <category android:name="android.intent.category.LAUNCHER"/>
                    
      </intent-filter>
              
    </activity>
            
    <meta-data android:name="android.support.VERSION" android:value="26.1.0"/>
            
    <meta-data android:name="android.arch.lifecycle.VERSION" android:value="27.0.0-SNAPSHOT"/>
        
  </application>
  
</manifest>`
