package uploaders

import (
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-steplib/steps-xcode-test/pretty"
)

func Test_fileName(t *testing.T) {
	tests := []struct {
		name string
		pth  string
		want string
	}{
		{
			name: "Does not modify path if does not have -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
			want: "app-demo-debug",
		},
		{
			name: "Trims -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-bitrise-signed.apk",
			want: "app-demo-debug",
		},
		{
			name: "Trims -unsigned suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-unsigned.apk",
			want: "app-demo-debug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileName(tt.pth); got != tt.want {
				t.Errorf("fileName() = %v, want %v", got, tt.want)
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

func Test_mapBuildArtifacts(t *testing.T) {
	tests := []struct {
		name string
		pths []string
		want BuildArtifactsMap
	}{
		{
			name: "APK split by density and abi",
			pths: []string{
				"app-arm64-v8a-debug.apk",
				"app-hdpiArmeabi-v7a-debug.apk",
				"app-mdpiX86-debug.apk",
				"app-xhdpiX86_64-debug.apk",
			},
			want: BuildArtifactsMap{
				"app": map[string]map[string][]string{
					"debug": map[string][]string{
						"": []string{
							"app-arm64-v8a-debug.apk",
							"app-hdpiArmeabi-v7a-debug.apk",
							"app-mdpiX86-debug.apk",
							"app-xhdpiX86_64-debug.apk",
						},
					},
				},
			},
		},
		{
			name: " -bitrise-signed and -unsigned apks",
			pths: []string{
				"app-debug-unsigned.apk",
				"app-debug-bitrise-signed.apk",
			},
			want: BuildArtifactsMap{
				"app": map[string]map[string][]string{
					"debug": map[string][]string{
						"": []string{
							"app-debug-unsigned.apk",
						},
					},
				},
			},
		},
		{
			name: " -bitrise-signed and -unsigned apks",
			pths: []string{
				"app-demo-debug-unsigned.apk",
				"app-demo-debug-bitrise-signed.apk",
			},
			want: BuildArtifactsMap{
				"app": map[string]map[string][]string{
					"debug": map[string][]string{
						"demo": []string{
							"app-demo-debug-unsigned.apk",
						},
					},
				},
			},
		},
		{
			name: "APK split by density and 2 flavours",
			pths: []string{
				"app-minApi21-full-hdpi-debug.apk",
				"app-minApi21-full-ldpi-debug.apk",
				"app-minApi21-full-mdpi-debug.apk",
				"app-minApi21-demo-universal-debug.apk",
				"app-minApi21-demo-xhdpi-debug.apk",
				"app-minApi21-demo-xxhdpi-debug.apk",
				"app-minApi21-demo-xxxhdpi-debug.apk",
			},
			want: BuildArtifactsMap{
				"app": map[string]map[string][]string{
					"debug": map[string][]string{
						"minApi21-demo": []string{
							"app-minApi21-demo-universal-debug.apk",
							"app-minApi21-demo-xhdpi-debug.apk",
							"app-minApi21-demo-xxhdpi-debug.apk",
							"app-minApi21-demo-xxxhdpi-debug.apk",
						},
						"minApi21-full": []string{
							"app-minApi21-full-hdpi-debug.apk",
							"app-minApi21-full-ldpi-debug.apk",
							"app-minApi21-full-mdpi-debug.apk",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapBuildArtifacts(tt.pths)

			if len(tt.want) != len(got) {
				t.Errorf("%v does not equal %v", pretty.Object(tt.want), pretty.Object(got))
				return
			}

			for wantModule, wantModuleArtifacts := range tt.want {
				gotModuleArtifacts := got[wantModule]

				if len(gotModuleArtifacts) != len(wantModuleArtifacts) {
					t.Errorf("%v does not equal %v", pretty.Object(tt.want), pretty.Object(got))
					return
				}

				for wantBuildType, wantBuildTypeArtifacts := range wantModuleArtifacts {
					gotBuildTypeArtifacts := gotModuleArtifacts[wantBuildType]

					if len(gotBuildTypeArtifacts) != len(wantBuildTypeArtifacts) {
						t.Errorf("%v does not equal %v", pretty.Object(tt.want), pretty.Object(got))
						return
					}

					if !compareMapStringStringSlice(wantBuildTypeArtifacts, gotBuildTypeArtifacts) {
						t.Errorf("%v does not equal %v", pretty.Object(tt.want), pretty.Object(got))
						return
					}
				}
			}
		})
	}
}

func comparseSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, e := range a {
		if !sliceutil.IsStringInSlice(e, b) {
			return false
		}
	}
	return true
}

func compareMapStringStringSlice(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}

	for keyA, valueA := range a {
		valueB, ok := b[keyA]
		if !ok {
			return false
		}

		if !comparseSlice(valueA, valueB) {
			return false
		}
	}

	return true
}

func Test_splitMeta(t *testing.T) {
	tests := []struct {
		name    string
		pth     string
		pths    []string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Split artifacts",
			pth:  "app-mdpiX86-debug.apk",
			pths: []string{
				"app-arm64-v8a-debug.apk",
				"app-hdpiArmeabi-v7a-debug.apk",
				"app-mdpiX86-debug.apk",
				"app-xhdpiX86_64-debug.apk",
			},
			want: map[string]interface{}{
				"split": []string{
					"app-arm64-v8a-debug.apk",
					"app-hdpiArmeabi-v7a-debug.apk",
					"app-mdpiX86-debug.apk",
					"app-xhdpiX86_64-debug.apk",
				},
				"include":   true,
				"universal": false,
			},
			wantErr: false,
		},
		{
			name: "Split artifacts with universal apk",
			pth:  "app-minApi21-demo-universal-debug.apk",
			pths: []string{
				"app-minApi21-demo-universal-debug.apk",
				"app-minApi21-demo-xhdpi-debug.apk",
				"app-minApi21-demo-xxhdpi-debug.apk",
				"app-minApi21-demo-xxxhdpi-debug.apk",
			},
			want: map[string]interface{}{
				"split": []string{
					"app-minApi21-demo-universal-debug.apk",
					"app-minApi21-demo-xhdpi-debug.apk",
					"app-minApi21-demo-xxhdpi-debug.apk",
					"app-minApi21-demo-xxxhdpi-debug.apk",
				},
				"include":   true,
				"universal": true,
			},
			wantErr: false,
		},
		{
			name: "Split artifacts with bitrise signed apk",
			pth:  "app-minApi21-demo-xhdpi-debug-bitrise-signed.apk",
			pths: []string{
				"app-minApi21-demo-universal-debug.apk",
				"app-minApi21-demo-xhdpi-debug.apk",
				"app-minApi21-demo-xxhdpi-debug.apk",
				"app-minApi21-demo-xxxhdpi-debug.apk",
				"app-minApi21-demo-xhdpi-debug-bitrise-signed.apk",
			},
			want: map[string]interface{}{
				"split": []string{
					"app-minApi21-demo-universal-debug.apk",
					"app-minApi21-demo-xhdpi-debug.apk",
					"app-minApi21-demo-xxhdpi-debug.apk",
					"app-minApi21-demo-xxxhdpi-debug.apk",
				},
				"include":   false,
				"universal": false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitMeta(tt.pth, tt.pths)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}
