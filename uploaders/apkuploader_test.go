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
