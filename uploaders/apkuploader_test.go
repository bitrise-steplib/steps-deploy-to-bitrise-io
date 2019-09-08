package uploaders

import (
	"testing"
)

func Test_filterPackageInfos(t *testing.T) {

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
			got, got1, got2 := filterPackageInfos(tt.aaptOut)
			if got != tt.want {
				t.Errorf("filterPackageInfos() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("filterPackageInfos() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("filterPackageInfos() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_trimBitriseSignedSuffix(t *testing.T) {
	tests := []struct {
		name string
		pth  string
		want string
	}{
		{
			name: "Does not modify path if does not have -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
			want: "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
		},
		{
			name: "Trims -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-bitrise-signed.apk",
			want: "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimBitriseSignedSuffix(tt.pth); got != tt.want {
				t.Errorf("trimBitriseSignedSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseAppPath(t *testing.T) {
	tests := []struct {
		name               string
		pth                string
		wantModule         string
		wantProductFlavour string
		wantBuildType      string
	}{
		{
			name:               "Parses apk path with Product Flavour",
			pth:                "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
			wantModule:         "app",
			wantProductFlavour: "demo",
			wantBuildType:      "debug",
		},
		{
			name:               "Parses apk path without Product Flavour",
			pth:                "$BITRISE_DEPLOY_DIR/app-debug.apk",
			wantModule:         "app",
			wantProductFlavour: "",
			wantBuildType:      "debug",
		},
		{
			name:               "Parses aab path with -bitrise-signed suffix",
			pth:                "$BITRISE_DEPLOY_DIR/app-demo-debug-bitrise-signed.apk",
			wantModule:         "app",
			wantProductFlavour: "demo",
			wantBuildType:      "debug",
		},
		{
			name:               "Returns empty for custom apk path",
			pth:                "$BITRISE_DEPLOY_DIR/custom.apk",
			wantModule:         "",
			wantProductFlavour: "",
			wantBuildType:      "",
		},
		{
			name:               "Parses ABI split apk path",
			pth:                "$BITRISE_SOURCE_DIR/app-arm64-v8a-debug.apk",
			wantModule:         "app",
			wantProductFlavour: "arm64-v8a",
			wantBuildType:      "debug",
		},
		{
			name:               "Parses 2 flavour dimensions, screen density split",
			pth:                "$BITRISE_SOURCE_DIR/app-minApi21-demo-hdpi-debug.apk",
			wantModule:         "app",
			wantProductFlavour: "minApi21-demo-hdpi",
			wantBuildType:      "debug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModule, gotProductFlavour, gotBuildType := parseAppPath(tt.pth)
			if gotModule != tt.wantModule {
				t.Errorf("parseAppPath() gotModule = %v, want %v", gotModule, tt.wantModule)
			}
			if gotProductFlavour != tt.wantProductFlavour {
				t.Errorf("parseAppPath() gotProductFlavour = %v, want %v", gotProductFlavour, tt.wantProductFlavour)
			}
			if gotBuildType != tt.wantBuildType {
				t.Errorf("parseAppPath() gotBuildType = %v, want %v", gotBuildType, tt.wantBuildType)
			}
		})
	}
}
