package common

// Utm50Driver are inherited from Utm47Driver.
// UTM 5.0 keeps the same scripting interface as 4.7 (qemu additional
// arguments, displays, serial ports, update configuration).
type Utm50Driver struct {
	Utm47Driver
}
