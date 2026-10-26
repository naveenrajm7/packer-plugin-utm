package common

import "testing"

func TestDriverForVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		wantV5  bool
		wantErr bool
	}{
		{"4.5.4", false, false},
		{"4.7.5", false, false},
		{"5.0.8", true, false},
		{"5.1.0", true, false},
		{"6.0.0", false, true},
		{"invalid", false, true},
	} {
		driver, err := driverForVersion(tc.version, "/usr/bin/utmctl")
		if (err != nil) != tc.wantErr {
			t.Errorf("version %s: error = %v", tc.version, err)
			continue
		}
		if !tc.wantErr {
			_, isV5 := driver.(*Utm50Driver)
			if isV5 != tc.wantV5 {
				t.Errorf("version %s: expected v5 driver = %v, got %T", tc.version, tc.wantV5, driver)
			}
		}
	}
}
