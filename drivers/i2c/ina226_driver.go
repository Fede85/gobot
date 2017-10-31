package i2c

// INA226Driver is a driver for the Texas Instrument INA226 IC.  The INA226 is a bi-directional
// shunt current and power monitor with I2C/SMBUS interface.
//
// INA226 datasheet: http://www.ti.com/product/INA226

import (
	"fmt"
	"math"

	"gobot.io/x/gobot"
)

const ina226I2CAddress = 0x40

// Registers address map
const (
	CONFIG_REG         byte = 0x00
	SHUNTVOLTAGE_REG   byte = 0x01
	BUSVOLTAGE_REG     byte = 0x02
	POWER_REG          byte = 0x03
	CURRENT_REG        byte = 0x04
	CALIBRATION_REG    byte = 0x05
	MASKENABLE_REG     byte = 0x06
	ALERTLIMIT_REG     byte = 0x07
	MANUFACTURERID_REG byte = 0xFE
	DIEID_REG          byte = 0xFF
)

// Configuration register helper constants
//
//   15   14  13  12    11     10     9       8       7       6      5      4      3      2     1     0
//  _____ ___ ___ ___ ______ ______ ______ _______ _______ _______ ______ ______ ______ _____ _____ _____
// |     |   |   |   |      |      |      |       |       |       |      |      |      |     |     |     |
// | RST | - | - | - | AVG2 | AVG1 | AVG0 |VBUSCT2|VBUSCT1|VBUSCT0|VSHCT2|VSHCT1|VSHCT0|MODE3|MODE2|MODE1|
// |_____|__ |___|___|______|______|______|_______|_______|_______|______|______|______|_____|_____|_____|

const (
	// MODE: operating mode (3-bit)
	INA226_MODE_POWER_DOWN     uint16 = 0x00
	INA226_MODE_SHUNT_TRIG     uint16 = 0x01
	INA226_MODE_BUS_TRIG       uint16 = 0x02
	INA226_MODE_SHUNT_BUS_TRIG uint16 = 0x03
	INA226_MODE_ADC_OFF        uint16 = 0x04
	INA226_MODE_SHUNT_CONT     uint16 = 0x05
	INA226_MODE_BUS_CONT       uint16 = 0x06
	INA226_MODE_SHUNT_BUS_CONT uint16 = 0x07

	// VSHCT: shunt voltage conversion time (3-bit)
	INA226_SHUNT_CONV_TIME_140US  uint16 = 0x00 << 3
	INA226_SHUNT_CONV_TIME_204US  uint16 = 0x01 << 3
	INA226_SHUNT_CONV_TIME_332US  uint16 = 0x02 << 3
	INA226_SHUNT_CONV_TIME_588US  uint16 = 0x03 << 3
	INA226_SHUNT_CONV_TIME_1100US uint16 = 0x04 << 3
	INA226_SHUNT_CONV_TIME_2116US uint16 = 0x05 << 3
	INA226_SHUNT_CONV_TIME_4156US uint16 = 0x06 << 3
	INA226_SHUNT_CONV_TIME_8244US uint16 = 0x07 << 3

	// VBUSCT: bus voltage conversion time (3-bit)
	INA226_BUS_CONV_TIME_140US  uint16 = 0x00 << 6
	INA226_BUS_CONV_TIME_204US  uint16 = 0x01 << 6
	INA226_BUS_CONV_TIME_332US  uint16 = 0x02 << 6
	INA226_BUS_CONV_TIME_588US  uint16 = 0x03 << 6
	INA226_BUS_CONV_TIME_1100US uint16 = 0x04 << 6
	INA226_BUS_CONV_TIME_2116US uint16 = 0x05 << 6
	INA226_BUS_CONV_TIME_4156US uint16 = 0x06 << 6
	INA226_BUS_CONV_TIME_8244US uint16 = 0x07 << 6

	// AVG: averaging mode (3-bit)
	INA226_AVERAGES_1    uint16 = 0x00 << 9
	INA226_AVERAGES_4    uint16 = 0x01 << 9
	INA226_AVERAGES_16   uint16 = 0x02 << 9
	INA226_AVERAGES_64   uint16 = 0x03 << 9
	INA226_AVERAGES_128  uint16 = 0x04 << 9
	INA226_AVERAGES_256  uint16 = 0x05 << 9
	INA226_AVERAGES_512  uint16 = 0x06 << 9
	INA226_AVERAGES_1024 uint16 = 0x07 << 9

	// RST bit
	INA226_RST uint16 = 0x01 << 15
)

type LoadSet struct {
	rShunt, iMax, vBusMax, vShuntMax float64
}

type INA226Driver struct {
	name       string
	connector  Connector
	connection Connection
	Config
	loadSet LoadSet
	halt    chan bool
}

func NewINA226Driver(c Connector, options ...func(Config)) *INA226Driver {
	i := &INA226Driver{
		name:      gobot.DefaultName("INA226"),
		connector: c,
		Config:    NewConfig(),
	}

	for _, option := range options {
		option(i)
	}

	return i
}

// Name returns the name of the device.
func (i *INA226Driver) Name() string {
	return i.name
}

// SetName sets the name of the device.
func (i *INA226Driver) SetName(name string) {
	i.name = name
}

// Connection returns the connection of the device.
func (i *INA226Driver) Connection() gobot.Connection {
	return i.connector.(gobot.Connection)
}

// Start initializes the INA3221
func (i *INA226Driver) Start() error {
	var err error
	bus := i.GetBusOrDefault(i.connector.GetDefaultBus())
	address := i.GetAddressOrDefault(int(ina226I2CAddress))

	if i.connection, err = i.connector.GetConnection(address, bus); err != nil {
		return err
	}

	i.Configure()
	return nil
}

// Halt halts the device.
func (i *INA226Driver) Halt() error {
	return nil
}

func wordToByteArray(w uint16) []byte {
	buf := make([]byte, 2)
	buf[0] = byte(w >> 8)
	buf[1] = byte(w)
	return buf
}

func (i *INA226Driver) Configure(confs ...uint16) error {
	var configuration uint16

	for _, conf := range confs {
		configuration |= conf
	}
	var buf []byte
	buf = append(buf, CONFIG_REG)
	buf = append(buf, wordToByteArray(configuration)...)

	_, err := i.connection.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (i *INA226Driver) Calibrate(rShuntValue float64, iMaxValue float64) error {
	i.loadSet.rShunt = rShuntValue
	i.loadSet.iMax = iMaxValue

	currentLSB := i.loadSet.iMax / 32768
	currentLSB *= 1000000 // transform to micro Ampere
	// As described in the datasheet to simplify calculation we should approximate the current LSB number
	// the method used is following described:
	// first extract from the currentLSB normalized notation only the mantissa
	currentLSB_mantissa := currentLSB / (math.Pow(10, math.Floor(math.Log10(currentLSB))))
	// then apply the ceiling function and multiply for the exponent
	currentLSB_approx := math.Ceil(currentLSB_mantissa) * math.Pow(10, math.Floor((math.Log10(currentLSB))))
	currentLSB = currentLSB_approx / 1000000 //transform back to Ampere

	calibrationValue := uint16((0.00512) / (currentLSB * i.loadSet.rShunt))

	var buf []byte
	buf = append(buf, CALIBRATION_REG)
	buf = append(buf, wordToByteArray(calibrationValue)...)

	_, err := i.connection.Write(buf)
	if err != nil {
		return err
	}
	return nil

}

// Reset set the reset bit in the configure register and reset the ic.
func (i *INA226Driver) Reset() error {
	var buf []byte
	buf = append(buf, CONFIG_REG)
	buf = append(buf, wordToByteArray(INA226_RST)...)
	_, err := i.connection.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (i *INA226Driver) readRegister16(reg byte) (uint16, error) {
	// send request to register
	_, err := i.connection.Write([]byte{reg})
	if err != nil {
		return 0, err
	}
	//read the 16 bit register content
	buf := make([]byte, 2)
	_, err = i.connection.Read(buf)
	if err != nil {
		return 0, err
	}

	value := uint16(buf[0])<<8 | uint16(buf[1])
	return value, nil
}

func (i *INA226Driver) readConfigurationRegister() (uint16, error) {
	confReg, err := i.readRegister16(CONFIG_REG)
	if err != nil {
		return 0, err
	}
	return confReg, nil
}

func (i *INA226Driver) readCalibrationRegister() (uint16, error) {
	calReg, err := i.readRegister16(CALIBRATION_REG)
	if err != nil {
		return 0, err
	}
	return calReg, nil
}

func (i *INA226Driver) CurrentResolution() (float64, error) {
	if i.loadSet.rShunt <= 0.0 {
		return 0, fmt.Errorf("rShunt value: %f is not correct. Must be greater than 0", i.loadSet.rShunt)
	}
	calibration, err := i.readCalibrationRegister()
	if err != nil {
		return 0, err
	}
	return 0.00512 / (float64(calibration) * i.loadSet.rShunt), nil
}

func (i *INA226Driver) ReadManufacturerRegister() (uint16, error) {
	confReg, err := i.readRegister16(MANUFACTURERID_REG)
	if err != nil {
		return 0, err
	}
	return confReg, nil
}

func (i *INA226Driver) ReadBusVoltage() (float64, error) {
	voltage, err := i.readRegister16(BUSVOLTAGE_REG)
	if err != nil {
		return 0, err
	}
	return float64(voltage) * 1.25, nil
}

func (i *INA226Driver) ReadShuntVoltage() (float64, error) {
	voltage, err := i.readRegister16(SHUNTVOLTAGE_REG)
	if err != nil {
		return 0, err
	}
	return float64(int16(voltage)) * 0.0025, nil
}

func (i *INA226Driver) ReadShuntCurrentRegister() (int16, error) {
	currentRaw, err := i.readRegister16(CURRENT_REG)
	if err != nil {
		return 0, err
	}
	return int16(currentRaw), nil
}

func (i *INA226Driver) ReadShuntCurrent() (float64, error) {
	currentRaw, err := i.ReadShuntCurrentRegister()
	if err != nil {
		return 0, err
	}

	currentResolution, err := i.CurrentResolution()
	if err != nil {
		return 0, err
	}

	return float64(currentRaw) * currentResolution, nil
}
