package rapl

import "math"

//parsers for turning the raw MSR uint into a struct

//Handle the MSR_[DOMAIN]_POWER_LIMIT MSR
func parsePowerLimit(msr uint64, units RAPLPowerUnit, singleLimit bool) RAPLPowerLimit {

	var powerLimit RAPLPowerLimit

	powerLimit.Limit1.PowerLimit = float64(msr&0x7fff) * units.PowerUnits
	powerLimit.Limit1.EnableLimit = ((msr >> 15) & 1) == 1
	powerLimit.Limit1.ClampingLimit = ((msr >> 16) & 1) == 1
	powerLimit.Limit1.TimeWindowLimit = parseTimeWindowLimit((msr>>17)&0x7f, units.TimeUnits)

	if singleLimit {
		powerLimit.Lock = ((msr >> 32) & 1) == 1
		return powerLimit

	}
	powerLimit.Limit2.PowerLimit = float64((msr>>32)&0x7fff) * units.PowerUnits
	powerLimit.Limit2.EnableLimit = ((msr >> 47) & 1) == 1
	powerLimit.Limit2.ClampingLimit = ((msr >> 48) & 1) == 1
	powerLimit.Limit2.TimeWindowLimit = parseTimeWindowLimit((msr>>49)&0x7f, units.TimeUnits)

	powerLimit.Lock = ((msr >> 63) & 1) == 1

	return powerLimit
}

//This equation is a pain, so we'll make our own function for it.
func parseTimeWindowLimit(rawLimit uint64, timeMult float64) float64 {

	//This is taken from the Intel SDM, Vol 3B 14.9.3
	/*
		Time limit = 2^Y * (1.0 + Z/4.0) * Time_Unit

		Here “Y” is the unsigned integer value represented by bits 21:17,
		“Z” is an unsigned integer represented by bits 23:22.
		“Time_Unit” is specified by the “Time Units” field of MSR_RAPL_POWER_UNIT.
	*/
	y := float64(rawLimit & 0x1f)
	z := float64(rawLimit >> 5 & 0x3)
	return math.Pow(2, y) * (1.0 + (z / 4.0)) * timeMult

}

//handle the MSR_RAPL_POWER_UNIT MSR
func parsePowerUnit(msr uint64) RAPLPowerUnit {

	var powerUnit RAPLPowerUnit
	//The values from the MSR are treated as registered according to the SDM
	powerUnit.PowerUnits = 1 / math.Pow(2, float64(msr&0xf))
	powerUnit.EnergyStatusUnits = 1 / math.Pow(2, float64((msr>>8)&0x1f))
	powerUnit.TimeUnits = 1 / math.Pow(2, float64((msr>>16)&0xf))

	return powerUnit
}

func parsePowerInfo(msr uint64, units RAPLPowerUnit) RAPLPowerInfo {

	var powerInfo RAPLPowerInfo
	powerInfo.ThermalSpecPower = float64(msr&0x7fff) * units.PowerUnits
	powerInfo.MinPower = float64((msr>>16)&0x7fff) * units.PowerUnits
	powerInfo.MaxPower = float64((msr>>32)&0x7fff) * units.PowerUnits
	powerInfo.MaxTimeWindow = float64((msr>>48)&0x3f) * units.TimeUnits

	return powerInfo
}
