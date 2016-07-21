package calculator

type CalculatorFactory interface {
	//NewBlkioCalculator(old BlkioData, new BlkioData) BlkioCalculator
	NewCPUCalculator(old CPUData, new CPUData) CPUCalculator
	//NewNetworkCalculator(old NetworkData, new NetworkData) NetworkCalculator
}

type CalculatorFactoryImpl struct {
}

/*func (c CalculatorFactoryImpl) NewBlkioCalculator(old BlkioData, new BlkioData) BlkioCalculator {
	return BlkioCalculatorImpl{Old: old, New: new}
}



func (c CalculatorFactoryImpl) NewNetworkCalculator(old NetworkData, new NetworkData) NetworkCalculator {
	return NetworkCalculatorImpl{old: old, new: new}
}
*/
func (c CalculatorFactoryImpl) NewCPUCalculator(old CPUData, new CPUData) CPUCalculator {
	return CPUCalculatorImpl{Old: old, New: new}
}