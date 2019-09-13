package androidartifact

import (
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-steplib/steps-xcode-test/pretty"
)

func TestParseArtifactPath(t *testing.T) {
	tests := []struct {
		name     string
		pth      string
		wantInfo ArtifactSigningInfo
		wantBase string
	}{
		{
			name: "Does not modify path if does not have -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
			wantInfo: ArtifactSigningInfo{
				Unsigned:      false,
				BitriseSigned: false,
			},
			wantBase: "app-demo-debug",
		},
		{
			name: "Trims -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-bitrise-signed.apk",
			wantInfo: ArtifactSigningInfo{
				Unsigned:      false,
				BitriseSigned: true,
			},
			wantBase: "app-demo-debug",
		},
		{
			name: "Trims -unsigned suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-unsigned.apk",
			wantInfo: ArtifactSigningInfo{
				Unsigned:      true,
				BitriseSigned: false,
			},
			wantBase: "app-demo-debug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInfo, gotBase := parseSigningInfo(tt.pth)
			if !reflect.DeepEqual(gotInfo, tt.wantInfo) {
				t.Errorf("parseSigningInfo() = %v, want %v", pretty.Object(gotInfo), pretty.Object(tt.wantInfo))
			}

			if gotBase != tt.wantBase {
				t.Errorf("parseSigningInfo() = %v, want %v", gotBase, tt.wantBase)
			}
		})
	}
}

func Test_parseArtifactInfo(t *testing.T) {
	tests := []struct {
		name string
		pth  string
		want ArtifactInfo
	}{
		{
			name: "Parses apk path with Product Flavour",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug.apk",
			want: ArtifactInfo{
				Module:         "app",
				ProductFlavour: "demo",
				BuildType:      "debug",
			},
		},
		{
			name: "Parses apk path without Product Flavour",
			pth:  "$BITRISE_DEPLOY_DIR/app-debug.apk",
			want: ArtifactInfo{
				Module:         "app",
				ProductFlavour: "",
				BuildType:      "debug",
			},
		},
		{
			name: "Parses aab path with -bitrise-signed suffix",
			pth:  "$BITRISE_DEPLOY_DIR/app-demo-debug-bitrise-signed.aab",
			want: ArtifactInfo{
				Module:         "app",
				ProductFlavour: "demo",
				BuildType:      "debug",
				SigningInfo: ArtifactSigningInfo{
					Unsigned:      false,
					BitriseSigned: true,
				},
			},
		},
		{
			name: "Returns empty for custom apk path",
			pth:  "$BITRISE_DEPLOY_DIR/custom.apk",
			want: ArtifactInfo{
				Module:         "",
				ProductFlavour: "",
				BuildType:      "",
			},
		},
		{
			name: "Parses ABI split apk path",
			pth:  "$BITRISE_SOURCE_DIR/app-arm64-v8a-debug.apk",
			want: ArtifactInfo{
				Module:         "app",
				ProductFlavour: "",
				BuildType:      "debug",
				SplitInfo: ArtifactSplitInfo{
					SplitParams: []string{"arm64-v8a"},
				},
			},
		},
		{
			name: "Parses 2 flavour dimensions, screen density split",
			pth:  "$BITRISE_SOURCE_DIR/app-minApi21-demo-hdpi-debug.apk",
			want: ArtifactInfo{
				Module:         "app",
				ProductFlavour: "minApi21-demo",
				BuildType:      "debug",
				SplitInfo: ArtifactSplitInfo{
					SplitParams: []string{"hdpi"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseArtifactPath(tt.pth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseArtifactPath() = %v, want %v", pretty.Object(got), pretty.Object(tt.want))
			}
		})
	}
}

func Test_mapBuildArtifacts(t *testing.T) {
	tests := []struct {
		name string
		pths []string
		want ArtifactMap
	}{
		{
			name: "APK split by density and abi",
			pths: []string{
				"app-arm64-v8a-debug.apk",
				"app-hdpiArmeabi-v7a-debug.apk",
				"app-mdpiX86-debug.apk",
				"app-xhdpiX86_64-debug.apk",
			},
			want: ArtifactMap{
				"app": map[string]map[string]Artifact{
					"debug": map[string]Artifact{
						"": Artifact{
							Split: []string{
								"app-arm64-v8a-debug.apk",
								"app-hdpiArmeabi-v7a-debug.apk",
								"app-mdpiX86-debug.apk",
								"app-xhdpiX86_64-debug.apk",
							},
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
			want: ArtifactMap{
				"app": map[string]map[string]Artifact{
					"debug": map[string]Artifact{
						"": Artifact{
							APK: "app-debug-bitrise-signed.apk",
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
			want: ArtifactMap{
				"app": map[string]map[string]Artifact{
					"debug": map[string]Artifact{
						"demo": Artifact{
							APK: "app-demo-debug-bitrise-signed.apk",
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
			want: ArtifactMap{
				"app": map[string]map[string]Artifact{
					"debug": map[string]Artifact{
						"minApi21-demo": Artifact{
							Split: []string{
								"app-minApi21-demo-universal-debug.apk",
								"app-minApi21-demo-xhdpi-debug.apk",
								"app-minApi21-demo-xxhdpi-debug.apk",
								"app-minApi21-demo-xxxhdpi-debug.apk",
							},
							UniversalApk: "app-minApi21-demo-universal-debug.apk",
						},
						"minApi21-full": Artifact{
							Split: []string{
								"app-minApi21-full-hdpi-debug.apk",
								"app-minApi21-full-ldpi-debug.apk",
								"app-minApi21-full-mdpi-debug.apk",
							},
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

				for wantBuildType, wantBuildTypeArtifact := range wantModuleArtifacts {
					gotBuildTypeArtifact := gotModuleArtifacts[wantBuildType]

					if len(gotBuildTypeArtifact) != len(wantBuildTypeArtifact) {
						t.Errorf("%v does not equal %v", pretty.Object(tt.want), pretty.Object(got))
						return
					}

					if !reflect.DeepEqual(wantBuildTypeArtifact, gotBuildTypeArtifact) {
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

func TestCreateSplitArtifactMeta(t *testing.T) {
	tests := []struct {
		name    string
		pth     string
		pths    []string
		want    SplitArtifactMeta
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
			want: SplitArtifactMeta{
				Split: []string{
					"app-arm64-v8a-debug.apk",
					"app-hdpiArmeabi-v7a-debug.apk",
					"app-mdpiX86-debug.apk",
					"app-xhdpiX86_64-debug.apk",
				},
				UniversalApk: "",
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
			want: SplitArtifactMeta{
				Split: []string{
					"app-minApi21-demo-universal-debug.apk",
					"app-minApi21-demo-xhdpi-debug.apk",
					"app-minApi21-demo-xxhdpi-debug.apk",
					"app-minApi21-demo-xxxhdpi-debug.apk",
				},
				UniversalApk: "app-minApi21-demo-universal-debug.apk",
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
			want: SplitArtifactMeta{
				Split: []string{
					"app-minApi21-demo-universal-debug.apk",
					"app-minApi21-demo-xhdpi-debug.apk",
					"app-minApi21-demo-xxhdpi-debug.apk",
					"app-minApi21-demo-xxxhdpi-debug.apk",
				},
				UniversalApk: "app-minApi21-demo-universal-debug.apk",
			},
			wantErr: false,
		},
		{
			name: "Split apks with aab",
			pth:  "app-minApi21-demo-xhdpi-debug-bitrise-signed.apk",
			pths: []string{
				"app-minApi21-demo-universal-debug.apk",
				"app-minApi21-demo-xhdpi-debug.apk",
				"app-minApi21-demo-xxhdpi-debug.apk",
				"app-minApi21-demo-xxxhdpi-debug.apk",
				"app-minApi21-demo-xhdpi-debug-bitrise-signed.apk",
				"app-minApi21-demo-debug.aab",
			},
			want: SplitArtifactMeta{
				Split: []string{
					"app-minApi21-demo-universal-debug.apk",
					"app-minApi21-demo-xhdpi-debug.apk",
					"app-minApi21-demo-xxhdpi-debug.apk",
					"app-minApi21-demo-xxxhdpi-debug.apk",
				},
				UniversalApk: "app-minApi21-demo-universal-debug.apk",
				AAB:          "app-minApi21-demo-debug.aab",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateSplitArtifactMeta(tt.pth, tt.pths)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSplitArtifactMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSplitArtifactMeta() = %v, want %v", pretty.Object(got), pretty.Object(tt.want))
			}
		})
	}
}

func TestUniversalAPKBase(t *testing.T) {
	tests := []struct {
		name       string
		basedOnAAB string
		want       string
	}{
		{
			name:       "simple test",
			basedOnAAB: "app-release.aab",
			want:       "app-universal-release.apk",
		},
		{
			name:       "bitrise signed aab",
			basedOnAAB: "app-release-bitrise-signed.aab",
			want:       "app-universal-release-bitrise-signed.apk",
		},
		{
			name:       "2 flavours",
			basedOnAAB: "app-minApi21-demo-debug.aab",
			want:       "app-minApi21-demo-universal-debug.apk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UniversalAPKBase(tt.basedOnAAB); got != tt.want {
				t.Errorf("UniversalAPKBase() = %v, want %v", got, tt.want)
			}
		})
	}
}
