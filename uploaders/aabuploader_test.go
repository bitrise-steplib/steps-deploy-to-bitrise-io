package uploaders

import "testing"

func Test_renameUniversalAPK(t *testing.T) {
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
			if got := renameUniversalAPK(tt.basedOnAAB); got != tt.want {
				t.Errorf("renameUniversalAPK() = %v, want %v", got, tt.want)
			}
		})
	}
}
