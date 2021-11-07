package rapl

//DomainMSRs carries address locations for various MSR registers
type DomainMSRs struct {
	PowerLimit   int64
	EnergyStatus int64
	Policy       int64
	PerfStatus   int64
	PowerInfo    int64
}

// The various RAPL domains

//RAPLDomain is a string type that covers the various RAPL domains
type RAPLDomain struct {
	Mask uint
	Name string
	MSRs DomainMSRs
}

//Package is the RAPL domain for the CPU package
var Package = RAPLDomain{0x1, "Package", DomainMSRs{0x610, 0x611, 0x0, 0x613, 0x614}}

//DRAM is the RAPL domain for the DRAM
var DRAM = RAPLDomain{0x2, "DRAM", DomainMSRs{0x618, 0x619, 0x0, 0x61b, 0x61c}}

//PP0 is the RAPL domain for the processor core
var PP0 = RAPLDomain{0x4, "PP0", DomainMSRs{0x638, 0x639, 0x63a, 0x63b, 0x0}}

//PP1 is platform-dependant, although it usually refers to some uncore power plane
var PP1 = RAPLDomain{0x8, "PP1", DomainMSRs{0x640, 0x641, 0x642, 0x0, 0x0}}

//MSRPowerUnit specifies the MSR for the MSR_RAPL_POWER_UNIT register
const MSRPowerUnit int64 = 0x606

// struct defs

//PowerLimitSetting specifies a power limit for a given time window
type PowerLimitSetting struct {
	//Sets the average power usage limits in Watts
	PowerLimit float64
	//Enables or disables the power limit
	EnableLimit bool
	//If enabled, this allows RAPL to turn down the processor frequency below what the OS has requested
	ClampingLimit bool
	//The time window,in seconds, over which the RAPL limit will be measured
	TimeWindowLimit float64
}

//RAPLPowerLimit contains the data in the MSR_[DOMAIN]_POWER_LIMIT MSR
//This MSR containers two power limits. From the SDM:
//"Two power limits can be specified, corresponding to time windows of different sizes"
//"Each power limit provides independent clamping control that would permit the processor cores to go below OS-requested state to meet the power limits."
type RAPLPowerLimit struct {
	Limit1 PowerLimitSetting
	Limit2 PowerLimitSetting
	Lock   bool
}

//RAPLPowerUnit contains the data in the MSR_RAPL_POWER_UNIT MSR
type RAPLPowerUnit struct {
	//PowerUnits is a multiplier for power related information in watts
	PowerUnits float64
	//EnergyStatusUnits is a multiplier for energy related information in joules
	EnergyStatusUnits float64
	//TimeUnits is a multiplier for time related information in seconds
	TimeUnits float64
}

//RAPLPowerInfo contains the data from the MSR_[DOMAIN]_POWER_INFO MSR
type RAPLPowerInfo struct {
	ThermalSpecPower float64
	MinPower         float64
	MaxPower         float64
	MaxTimeWindow    float64
}
