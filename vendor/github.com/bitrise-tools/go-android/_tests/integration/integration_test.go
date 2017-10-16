package integration

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"os"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-tools/go-android/adbmanager"
	"github.com/bitrise-tools/go-android/avdmanager"
	"github.com/bitrise-tools/go-android/emulatormanager"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
	"github.com/stretchr/testify/require"
)

const (
	testEmulatorName = "test_emu_name"
	testEmulatorSkin = "768x1280"
)

type emulator struct {
	platform string
	tag      string
	abi      string
	runOptions []string
	isSupportedByLegacyStack bool
}

func TestIsLegacyAVDManager(t *testing.T) {
	_, err := avdmanager.IsLegacyAVDManager(os.Getenv("ANDROID_HOME"))
	require.NoError(t, err)
}

func TestCreateAndStartEmulator(t *testing.T) {
	isLegacyStack, _ := avdmanager.IsLegacyAVDManager(os.Getenv("ANDROID_HOME"))
	
	for _, emu := range getEmulatorConfigList() {
		if isLegacyStack && !emu.isSupportedByLegacyStack {
			continue
		}
		createEmulator(t, emu.platform, emu.tag, emu.abi)
		startEmulator(t, emu.runOptions)
	}
}

func getEmulatorConfigList() []emulator {
	googleApis25RanchuOptions := []string{"-kernel", os.Getenv("ANDROID_HOME") + "/system-images/android-25/google_apis/arm64-v8a/kernel-qemu"}
	return []emulator{
		emulator{platform: "android-24", tag: "google_apis", abi: "armeabi-v7a", runOptions: []string{}, isSupportedByLegacyStack: true},
		emulator{platform: "android-25", tag: "android-wear", abi: "armeabi-v7a", runOptions: []string{}, isSupportedByLegacyStack: true},
		emulator{platform: "android-23", tag: "android-tv", abi: "armeabi-v7a", runOptions: []string{}, isSupportedByLegacyStack: true},
		emulator{platform: "android-19", tag: "default", abi: "armeabi-v7a", runOptions:[]string{}, isSupportedByLegacyStack: true},
		emulator{platform: "android-17", tag: "default", abi: "mips", runOptions: []string{}, isSupportedByLegacyStack: true},
		emulator{platform: "android-25", tag: "google_apis", abi: "arm64-v8a", runOptions: googleApis25RanchuOptions, isSupportedByLegacyStack: false},
	}
}

func createEmulator(t *testing.T, platform string, tag string, abi string) {
	t.Logf("Create emulator: %s - %s - %s", platform, tag, abi)

	androidSdk, err := sdk.New(os.Getenv("ANDROID_HOME"))
	require.NoError(t, err)

	manager, err := sdkmanager.New(androidSdk)
	require.NoError(t, err)

	platformComponent := sdkcomponent.Platform{
		Version: platform,
	}

	platformInstalled, err := manager.IsInstalled(platformComponent)
	require.NoError(t, err)

	if !platformInstalled {
		t.Logf("Installing platform: %s", platform)

		installCmd := manager.InstallCommand(platformComponent)
		installCmd.SetStdin(strings.NewReader("y"))

		t.Logf("$ %s", installCmd.PrintableCommandArgs())

		out, err := installCmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		installed, err := manager.IsInstalled(platformComponent)
		require.NoError(t, err)
		require.Equal(t, true, installed)
	}

	systemImageComponent := sdkcomponent.SystemImage{
		Platform: platform,
		Tag:      tag,
		ABI:      abi,
	}

	systemImageInstalled, err := manager.IsInstalled(systemImageComponent)
	require.NoError(t, err)

	if !systemImageInstalled {
		t.Logf("Installing system image (platform: %s abi: %s tag: %s)", systemImageComponent.Platform, systemImageComponent.ABI, systemImageComponent.Tag)

		installCmd := manager.InstallCommand(systemImageComponent)
		installCmd.SetStdin(strings.NewReader("y"))

		t.Logf("$ %s", installCmd.PrintableCommandArgs())

		out, err := installCmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		installed, err := manager.IsInstalled(systemImageComponent)
		require.NoError(t, err)
		require.Equal(t, true, installed)
	}

	avdManager, err := avdmanager.New(androidSdk)
	require.NoError(t, err)

	cmd := avdManager.CreateAVDCommand(testEmulatorName, systemImageComponent)
	cmd.SetStdin(strings.NewReader("n"))

	t.Logf("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	require.NoError(t, err, out)
}

func startEmulator(t *testing.T, runOptions []string) {
	t.Logf("Start emulator")

	avdImages, err := listAVDImages()
	require.NoError(t, err)

	if !sliceutil.IsStringInSlice(testEmulatorName, avdImages) {
		require.FailNow(t, "No emulator found with name: " + testEmulatorName)
	}

	androidSdk, err := sdk.New(os.Getenv("ANDROID_HOME"))
	require.NoError(t, err)

	adb, err := adbmanager.New(androidSdk)
	require.NoError(t, err)

	deviceStateMap, err := runningDeviceInfos(*adb)
	require.NoError(t, err)

	emulator, err := emulatormanager.New(androidSdk)
	require.NoError(t, err)

	options := []string{"-no-boot-anim", "-no-window"}
        options = append(options, runOptions...)

	startEmulatorCommand := emulator.StartEmulatorCommand(testEmulatorName, testEmulatorSkin, options...)
	startEmulatorCmd := startEmulatorCommand.GetCmd()

	e := make(chan error)

	stdoutReader, err := startEmulatorCmd.StdoutPipe()
	require.NoError(t, err)

	outScanner := bufio.NewScanner(stdoutReader)
	go func() {
		for outScanner.Scan() {
			line := outScanner.Text()
			fmt.Println(line)
		}
	}()
	err = outScanner.Err()
	require.NoError(t, err)

	stderrReader, err := startEmulatorCmd.StderrPipe()
	require.NoError(t, err)

	errScanner := bufio.NewScanner(stderrReader)
	go func() {
		for errScanner.Scan() {
			line := errScanner.Text()
			log.Warnf(line)
		}
	}()
	err = errScanner.Err()
	require.NoError(t, err)

	serial := ""

	go func() {
		t.Logf("$ %s", command.PrintableCommandArgs(false, startEmulatorCmd.Args))

		if err := startEmulatorCommand.Run(); err != nil {
			e <- err
			return
		}
	}()

	go func() {
		t.Logf("Checking for started device serial...")
		for len(serial) == 0 {
			time.Sleep(5 * time.Second)

			currentDeviceStateMap, err := runningDeviceInfos(*adb)
			if err != nil {
				e <- err
				return
			}

			serial = currentlyStartedDeviceSerial(deviceStateMap, currentDeviceStateMap)
		}
		t.Logf("Started device serial: %s", serial)

		bootInProgress := true
		t.Log("Wait for emulator to boot...")
		for bootInProgress {
			time.Sleep(5 * time.Second)

			booted, err := adb.IsDeviceBooted(serial)
			if err != nil {
				e <- err
				return
			}

			bootInProgress = !booted
		}

		err := adb.UnlockDevice(serial)
		require.NoError(t, err)

		log.Donef("Emulator booted")

		e <- nil
	}()

	timeout := 600

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		err := startEmulatorCmd.Process.Kill()
		if err != nil {
			t.Logf("failed to kill process: %s", err)
		}
		require.FailNow(t, "Boot timed out...")

	case err := <-e:
		require.NoError(t, err)
	}

	err = startEmulatorCmd.Process.Kill()
	if err != nil {
		t.Logf("failed to kill process: %s", err)
	}
}

func listAVDImages() ([]string, error) {
	homeDir := pathutil.UserHomeDir()
	avdDir := filepath.Join(homeDir, ".android", "avd")

	avdImagePattern := filepath.Join(avdDir, "*.ini")
	avdImages, err := filepath.Glob(avdImagePattern)
	if err != nil {
		return []string{}, fmt.Errorf("glob failed with pattern (%s), error: %s", avdImagePattern, err)
	}

	avdImageNames := []string{}

	for _, avdImage := range avdImages {
		imageName := filepath.Base(avdImage)
		imageName = strings.TrimSuffix(imageName, filepath.Ext(avdImage))
		avdImageNames = append(avdImageNames, imageName)
	}

	return avdImageNames, nil
}

func avdImageDir(name string) string {
	homeDir := pathutil.UserHomeDir()
	return filepath.Join(homeDir, ".android", "avd", name+".avd")
}

func currentlyStartedDeviceSerial(alreadyRunningDeviceInfos, currentlyRunningDeviceInfos map[string]string) string {
	startedSerial := ""

	for serial := range currentlyRunningDeviceInfos {
		_, found := alreadyRunningDeviceInfos[serial]
		if !found {
			startedSerial = serial
			break
		}
	}

	if len(startedSerial) > 0 {
		state := currentlyRunningDeviceInfos[startedSerial]
		if state == "device" {
			return startedSerial
		}
	}

	return ""
}

func runningDeviceInfos(adb adbmanager.Model) (map[string]string, error) {
	cmd := adb.DevicesCmd()
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return map[string]string{}, fmt.Errorf("command failed, error: %s", err)
	}

	deviceListItemPattern := `^(?P<emulator>emulator-\d*)[\s+](?P<state>.*)`
	deviceListItemRegexp := regexp.MustCompile(deviceListItemPattern)

	deviceStateMap := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		matches := deviceListItemRegexp.FindStringSubmatch(line)
		if len(matches) == 3 {
			serial := matches[1]
			state := matches[2]

			deviceStateMap[serial] = state
		}

	}
	if scanner.Err() != nil {
		return map[string]string{}, fmt.Errorf("scanner failed, error: %s", err)
	}

	return deviceStateMap, nil
}
