package appstore

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"howett.net/plist"
)

func TestValidateLegacyPackagePlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		info     map[string]interface{}
		accept   bool
	}{
		{"iPhone accepts legacy declaration", PlatformIPhone, map[string]interface{}{"LSRequiresIPhoneOS": true}, true},
		{"iPad accepts legacy declaration", PlatformIPad, map[string]interface{}{"LSRequiresIPhoneOS": true}, true},
		{"missing declaration is rejected", PlatformIPhone, map[string]interface{}{}, false},
		{"false legacy declaration is rejected", PlatformIPhone, map[string]interface{}{"LSRequiresIPhoneOS": false}, false},
		{"empty explicit platforms are rejected", PlatformIPhone, map[string]interface{}{"LSRequiresIPhoneOS": true, "CFBundleSupportedPlatforms": []string{}}, false},
		{"explicit tvOS takes precedence", PlatformIPhone, map[string]interface{}{"LSRequiresIPhoneOS": true, "CFBundleSupportedPlatforms": []string{"AppleTVOS"}}, false},
		{"explicit watchOS takes precedence", PlatformIPad, map[string]interface{}{"LSRequiresIPhoneOS": true, "CFBundleSupportedPlatforms": []string{"WatchOS"}}, false},
		{"explicit visionOS takes precedence", PlatformIPhone, map[string]interface{}{"LSRequiresIPhoneOS": true, "CFBundleSupportedPlatforms": []string{"XROS"}}, false},
		{"legacy iOS does not imply tvOS", PlatformAppleTV, map[string]interface{}{"LSRequiresIPhoneOS": true}, false},
		{"legacy iOS does not imply watchOS", PlatformWatchOS, map[string]interface{}{"LSRequiresIPhoneOS": true}, false},
		{"legacy iOS does not imply visionOS", PlatformVisionOS, map[string]interface{}{"LSRequiresIPhoneOS": true}, false},
	}

	for _, format := range []struct {
		name  string
		value int
	}{{"XML", plist.XMLFormat}, {"binary", plist.BinaryFormat}} {
		t.Run(format.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					path := writeLegacyPlatformPackage(t, format.value, map[string]map[string]interface{}{
						"Payload/Test.app/Info.plist": tt.info,
					})
					err := (&appstore{}).validatePackagePlatform(path, tt.platform)

					if (err == nil) != tt.accept {
						t.Fatalf("validatePackagePlatform() error = %v, want accepted = %v", err, tt.accept)
					}
				})
			}
		})
	}
}

func TestValidateLegacyPackagePlatformIgnoresEmbeddedApps(t *testing.T) {
	path := writeLegacyPlatformPackage(t, plist.BinaryFormat, map[string]map[string]interface{}{
		"Payload/Test.app/Info.plist":                 {"CFBundleSupportedPlatforms": []string{"AppleTVOS"}},
		"Payload/Test.app/PlugIns/iOS.app/Info.plist": {"LSRequiresIPhoneOS": true},
	})
	if err := (&appstore{}).validatePackagePlatform(path, PlatformIPhone); err == nil {
		t.Fatal("accepted iOS support declared only by an embedded app")
	}
}

func writeLegacyPlatformPackage(t *testing.T, format int, entries map[string]map[string]interface{}) string {
	t.Helper()

	var archive bytes.Buffer

	writer := zip.NewWriter(&archive)
	for name, info := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}

		data, err := plist.Marshal(info, format)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "legacy.ipa")
	if err := os.WriteFile(path, archive.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	return path
}
