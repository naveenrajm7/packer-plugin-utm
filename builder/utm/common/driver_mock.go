package common

import "sync"

type DriverMock struct {
	sync.Mutex

	DeleteCalled bool
	DeleteName   string
	DeleteErr    error

	ExecuteOsaCalls  [][]string
	ExecuteOsaErrs   []error
	ExecuteOsaResult string

	ImportCalled bool
	ImportName   string
	ImportPath   string
	ImportErr    error

	IsRunningName   string
	IsRunningReturn bool
	IsRunningErr    error

	StopName string
	StopErr  error

	UtmctlCalls  [][]string
	UtmctlErrs   []error
	UtmctlResult string

	VerifyCalled bool
	VerifyErr    error

	VersionCalled bool
	VersionResult string
	VersionErr    error
}

func (d *DriverMock) Delete(name string) error {
	d.DeleteCalled = true
	d.DeleteName = name
	return d.DeleteErr
}

func (d *DriverMock) ExecuteOsaScript(command ...string) (string, error) {
	d.ExecuteOsaCalls = append(d.ExecuteOsaCalls, command)

	if len(d.ExecuteOsaErrs) >= len(d.ExecuteOsaCalls) {
		return "", d.ExecuteOsaErrs[len(d.ExecuteOsaCalls)-1]
	}
	return d.ExecuteOsaResult, nil
}

func (d *DriverMock) Import(name string, path string) error {
	d.ImportCalled = true
	d.ImportName = name
	d.ImportPath = path
	return d.ImportErr
}

func (d *DriverMock) IsRunning(name string) (bool, error) {
	d.Lock()
	defer d.Unlock()

	d.IsRunningName = name
	return d.IsRunningReturn, d.IsRunningErr
}

func (d *DriverMock) Stop(name string) error {
	d.StopName = name
	return d.StopErr
}

func (d *DriverMock) Utmctl(args ...string) (string, error) {
	d.UtmctlCalls = append(d.UtmctlCalls, args)

	if len(d.UtmctlErrs) >= len(d.UtmctlCalls) {
		return "", d.UtmctlErrs[len(d.UtmctlCalls)-1]
	}
	return d.UtmctlResult, nil
}

func (d *DriverMock) Verify() error {
	d.VerifyCalled = true
	return d.VerifyErr
}

func (d *DriverMock) Version() (string, error) {
	d.VersionCalled = true
	return d.VersionResult, d.VersionErr
}
