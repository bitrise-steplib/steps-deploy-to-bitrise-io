package main

import "testing"

func Test_findUniversalAPKPair(t *testing.T) {
	tests := []struct {
		name string
		aab  string
		apks []string
		want string
	}{
		{
			name: "Only universal apk can be the pair",
			aab:  "app-minApi21-demo-debug-bitrise-signed.aab",
			apks: []string{"app-minApi21-demo-debug.apk"},
			want: "",
		},
		{
			name: "Finds if aab is bitrise-signed",
			aab:  "app-minApi21-demo-debug-bitrise-signed.aab",
			apks: []string{"app-minApi21-demo-universal-debug.apk"},
			want: "app-minApi21-demo-universal-debug.apk",
		},
		{
			name: "Finds if aab is bitrise-signed, even if apk is unsigned",
			aab:  "app-minApi21-demo-debug-bitrise-signed.aab",
			apks: []string{"app-minApi21-demo-universal-debug-unsigned.apk"},
			want: "app-minApi21-demo-universal-debug-unsigned.apk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findUniversalAPKPair(tt.aab, tt.apks); got != tt.want {
				t.Errorf("findUniversalAPKPair() = %v, want %v", got, tt.want)
			}
		})
	}
}
