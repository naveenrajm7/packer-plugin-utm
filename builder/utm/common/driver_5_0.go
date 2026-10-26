package common

// Utm50Driver retains the scripting operations used by UTM 4.7 and accepts
// UTM 5's native utmctl for lifecycle and guest provisioning commands.
type Utm50Driver struct {
	Utm47Driver
}
