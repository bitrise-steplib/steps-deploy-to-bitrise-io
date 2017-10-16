package sdkcomponent

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Model ...
type Model interface {
	GetSDKStylePath() string
	GetLegacySDKStylePath() string
	InstallPathInAndroidHome() string
	InstallationIndicatorFile() string
}

// SDKTool ...
type SDKTool struct {
	SDKStylePath       string
	LegacySDKStylePath string
}

// GetSDKStylePath ...
func (component SDKTool) GetSDKStylePath() string {
	if component.SDKStylePath != "" {
		return component.SDKStylePath
	}
	return "tools"
}

// GetLegacySDKStylePath ...
func (component SDKTool) GetLegacySDKStylePath() string {
	if component.LegacySDKStylePath != "" {
		return component.LegacySDKStylePath
	}
	return "tools"
}

// InstallPathInAndroidHome ...
func (component SDKTool) InstallPathInAndroidHome() string {
	return "tools"
}

// InstallationIndicatorFile ...
func (component SDKTool) InstallationIndicatorFile() string {
	return ""
}

// BuildTool ...
type BuildTool struct {
	Version string

	SDKStylePath       string
	LegacySDKStylePath string
}

// GetSDKStylePath ...
func (component BuildTool) GetSDKStylePath() string {
	if component.SDKStylePath != "" {
		return component.SDKStylePath
	}
	return fmt.Sprintf("build-tools;%s", component.Version)
}

// GetLegacySDKStylePath ...
func (component BuildTool) GetLegacySDKStylePath() string {
	if component.LegacySDKStylePath != "" {
		return component.LegacySDKStylePath
	}
	return fmt.Sprintf("build-tools-%s", component.Version)
}

// InstallPathInAndroidHome ...
func (component BuildTool) InstallPathInAndroidHome() string {
	return filepath.Join("build-tools", component.Version)
}

// InstallationIndicatorFile ...
func (component BuildTool) InstallationIndicatorFile() string {
	return ""
}

// Platform ...
type Platform struct {
	Version string

	SDKStylePath       string
	LegacySDKStylePath string
}

// GetSDKStylePath ...
func (component Platform) GetSDKStylePath() string {
	if component.SDKStylePath != "" {
		return component.SDKStylePath
	}
	return fmt.Sprintf("platforms;%s", component.Version)
}

// GetLegacySDKStylePath ...
func (component Platform) GetLegacySDKStylePath() string {
	if component.LegacySDKStylePath != "" {
		return component.LegacySDKStylePath
	}
	return component.Version
}

// InstallPathInAndroidHome ...
func (component Platform) InstallPathInAndroidHome() string {
	return filepath.Join("platforms", component.Version)
}

// InstallationIndicatorFile ...
func (component Platform) InstallationIndicatorFile() string {
	return ""
}

// SystemImage ...
type SystemImage struct {
	Platform string
	ABI      string
	Tag      string

	SDKStylePath       string
	LegacySDKStylePath string
}

// GetSDKStylePath ...
func (component SystemImage) GetSDKStylePath() string {
	if component.SDKStylePath != "" {
		return component.SDKStylePath
	}

	tag := "default"
	if component.Tag != "" {
		tag = component.Tag
	}

	return fmt.Sprintf("system-images;%s;%s;%s", component.Platform, tag, component.ABI)
}

// GetLegacySDKStylePath ...
func (component SystemImage) GetLegacySDKStylePath() string {
	if component.LegacySDKStylePath != "" {
		return component.LegacySDKStylePath
	}

	platform := component.Platform
	if component.Tag != "" && component.Tag != "default" {
		split := strings.Split(component.Platform, "-")
		if len(split) == 2 {
			platform = component.Tag + "-" + split[1]
		}
	}

	return fmt.Sprintf("sys-img-%s-%s", component.ABI, platform)
}

// InstallPathInAndroidHome ...
func (component SystemImage) InstallPathInAndroidHome() string {
	componentTag := "default"
	if component.Tag != "" {
		componentTag = component.Tag
	}

	return filepath.Join("system-images", component.Platform, componentTag, component.ABI)
}

// InstallationIndicatorFile ...
func (component SystemImage) InstallationIndicatorFile() string {
	return "system.img"
}

// Extras ...
type Extras struct {
	Provider    string
	PackageName string

	SDKStylePath       string
	LegacySDKStylePath string
}

// GooglePlayServicesInstallComponents ...
func GooglePlayServicesInstallComponents() []Extras {
	return []Extras{
		Extras{
			Provider:    "google",
			PackageName: "m2repository",
		},
		Extras{
			Provider:    "google",
			PackageName: "google_play_services",
		},
	}
}

// LegacyGooglePlayServicesInstallComponents ...
func LegacyGooglePlayServicesInstallComponents() []Extras {
	return []Extras{
		Extras{
			Provider:    "google",
			PackageName: "m2repository",
		},
		Extras{
			Provider:    "google",
			PackageName: "google_play_services",
		},
	}
}

// SupportLibraryInstallComponents ...
func SupportLibraryInstallComponents() []Extras {
	return []Extras{
		Extras{
			Provider:    "android",
			PackageName: "m2repository",
		},
		// Extras{
		// 	Provider:    "android",
		// 	PackageName: "support",
		// },
	}
}

// LegacySupportLibraryInstallComponents ...
func LegacySupportLibraryInstallComponents() []Extras {
	return []Extras{
		Extras{
			Provider:    "android",
			PackageName: "m2repository",
		},
	}
}

// GetSDKStylePath ...
func (component Extras) GetSDKStylePath() string {
	if component.SDKStylePath != "" {
		return component.SDKStylePath
	}

	return fmt.Sprintf("extras;%s;%s", component.Provider, component.PackageName)
}

// GetLegacySDKStylePath ...
func (component Extras) GetLegacySDKStylePath() string {
	if component.LegacySDKStylePath != "" {
		return component.LegacySDKStylePath
	}

	return fmt.Sprintf("extra-%s-%s", component.Provider, component.PackageName)
}

// InstallPathInAndroidHome ...
func (component Extras) InstallPathInAndroidHome() string {
	return filepath.Join("extras", component.Provider, component.PackageName)
}

// InstallationIndicatorFile ...
func (component Extras) InstallationIndicatorFile() string {
	return ""
}
