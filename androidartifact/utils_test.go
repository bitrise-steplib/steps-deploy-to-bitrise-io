package androidartifact

import (
	"testing"
)

func TestParsePackageInfos(t *testing.T) {
	tests := []struct {
		name           string
		aaptOut        string
		packageNameKey string
		packageName    string
		versionCode    string
		versionName    string
	}{
		{
			name:           "test_with_empty_platformBuildVersionName",
			aaptOut:        `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0' platformBuildVersionName=''`,
			packageNameKey: "name",
			packageName:    "com.example.birmachera.myapplication",
			versionCode:    "1",
			versionName:    "1.0",
		},
		{
			name:           "test_without_platformBuildVersionName",
			aaptOut:        `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0'`,
			packageNameKey: "name",
			packageName:    "com.example.birmachera.myapplication",
			versionCode:    "1",
			versionName:    "1.0",
		},
		{
			name:           "test_with_platformBuildVersionName",
			aaptOut:        `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0' platformBuildVersionName='3'`,
			packageNameKey: "name",
			packageName:    "com.example.birmachera.myapplication",
			versionCode:    "1",
			versionName:    "1.0",
		},
		{
			name:           "test_without_name",
			aaptOut:        `package: name='' versionCode='1' versionName='1.0' platformBuildVersionName='3'`,
			packageNameKey: "name",
			packageName:    "",
			versionCode:    "1",
			versionName:    "1.0",
		},
		{
			name:           "test_without_name_and_versionCode",
			aaptOut:        `package: name='' versionCode='' versionName='1.0' platformBuildVersionName='3'`,
			packageNameKey: "name",
			packageName:    "",
			versionCode:    "",
			versionName:    "1.0",
		},
		{
			name:           "test_without_name_and_versionCode_and_versionName",
			aaptOut:        `package: name='' versionCode='' versionName='' platformBuildVersionName='3'`,
			packageNameKey: "name",
			packageName:    "",
			versionCode:    "",
			versionName:    "",
		},
		{
			name:           "test_without_name_and_versionCode_and_versionName",
			aaptOut:        `package: name='' versionCode='2' versionName='' platformBuildVersionName='3'`,
			packageNameKey: "name",
			packageName:    "",
			versionCode:    "2",
			versionName:    "",
		},
		{
			name:           "test_aab_package_name_from_manifest",
			aaptOut:        testArtifactAndroidManifest,
			packageNameKey: "package",
			packageName:    "com.example.birmachera.myapplication",
			versionCode:    "1",
			versionName:    "1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageName, versionCode, versionName := ParsePackageInfo(tt.aaptOut, tt.packageNameKey)
			if packageName != tt.packageName {
				t.Errorf("packageName got = %v, want %v", packageName, tt.packageName)
			}
			if versionCode != tt.versionCode {
				t.Errorf("versionCode got = %v, want %v", versionCode, tt.versionCode)
			}
			if versionName != tt.versionName {
				t.Errorf("versionName got = %v, want %v", versionName, tt.versionName)
			}
		})
	}
}
