package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

var randSource rand.Source

func init() {
	randSource = rand.NewSource(time.Now().UnixNano())
}

// UTILS
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func num2Bin[T Integer](n T) string {
	var str string
	size := unsafe.Sizeof(n) * 8

	for i := 0; i < int(size); i++ {
		if (n>>i)&0x1 == 0x1 {
			str += "1"
		} else {
			str += "0"
		}
	}

	return str

}

func regFileAsBin(regFile RegisterFile) string {
	str := `{`
	str += fmt.Sprintf("\n\tPC: %s", num2Bin(regFile.PC))
	for i, v := range regFile.Registers {
		str += fmt.Sprintf("\n\tV[%X]: %s", i, num2Bin(v))
	}
	str += "\n"
	str += `}`

	return str
}

// Opcode
type Opcode struct {
	Value []byte
}

func (o Opcode) String() string {
	return fmt.Sprintf("%X", o.Value)
}

// Arguments
type OperationArgs struct {
	Address              uint16
	XRegIndex, YRegIndex uint8
	Value                uint16
}

func (a OperationArgs) String() string {
	return fmt.Sprintf(`
		{
			Address: %04X,
			XRegIndex: %0X, YRegIndex: %0X
			Value : %04X
		}`, a.Address,
		a.XRegIndex, a.YRegIndex,
		a.Value)
}

type ExecFunc func(RegisterFile, OperationArgs) RegisterFile

// Register File
type RegisterFile struct {
	Registers [16]byte // V[F] is the flags register
	PC        uint16
}

func NewRegisterFile() RegisterFile {
	return RegisterFile{
		Registers: [16]byte{},
		PC:        0x0000,
	}
}

func (rf RegisterFile) String() string {
	str := `{`
	str += fmt.Sprintf("\n\tPC: %04X", rf.PC)
	for i, v := range rf.Registers {
		str += fmt.Sprintf("\n\tV[%X]: %02X", i, v)
	}
	str += "\n"
	str += `}`

	return str
}

// EXTRACTORS
func extractXAndValue(opcode Opcode) *OperationArgs {
	var (
		args     *OperationArgs
		regIndex uint8
		value    uint16
	)

	regIndex = opcode.Value[0] & 0x0F
	value = uint16(opcode.Value[1])
	args = &OperationArgs{
		XRegIndex: regIndex,
		Value:     value,
	}

	return args
}

func extractAddress(opcode Opcode) *OperationArgs {
	var (
		address uint16
		args    *OperationArgs
	)

	address = uint16(opcode.Value[0] & 0x0F)
	address <<= 8
	address += uint16(opcode.Value[1])
	args = &OperationArgs{
		Address: address,
	}

	return args
}

func extractXAndY(opcode Opcode) *OperationArgs {
	var (
		regX uint8
		regY uint8
		args *OperationArgs
	)

	regX = opcode.Value[0] & 0x0F
	regY = (opcode.Value[1] & 0xF0) >> 4

	args = &OperationArgs{
		XRegIndex: regX,
		YRegIndex: regY,
	}

	return args
}

func Decode(opcode Opcode) (ExecFunc, *OperationArgs) {
	var args *OperationArgs
	signature := (opcode.Value[0] & 0xF0) >> 4
	args = &OperationArgs{}

	switch signature {
	case 0x0:
		return Halt, args
	case 0x1:
		args = extractAddress(opcode)

		return Jump2Addr, args
	case 0x3:
		args = extractXAndValue(opcode)

		return SkipNextInstrIf, args
	case 0x6:
		args = extractXAndValue(opcode)

		return AsignToReg, args
	case 0x7:
		args = extractXAndValue(opcode)

		return AddValueToReg, args
	case 0x8:
		opType := opcode.Value[1] & 0x0F
		args = extractXAndY(opcode)
		switch opType {
		case 0x0:
			return AsignReg2Reg, args
		case 0x1:
			return RegOrReg, args
		case 0x2:
			return RegAndReg, args
		case 0x3:
			return RegXorReg, args
		case 0x4:
			return RegPlusReg, args
		case 0x5:
			return RegMinusReg, args
		case 0x6:
			return RegRShift, args
		case 0x7:
			return RegMinusRegInv, args
		case 0xE:
			return RegLShift, args
		default:
			return Halt, args
		}

	case 0xC:
		args = extractXAndValue(opcode)
		return RandomNumAndReg, args

	default:
		return Halt, args
	}
}

// INTRUCTIONS FUNCTIONS
// 0000: HALT
func Halt(RegisterFile, OperationArgs) RegisterFile {
	// finish exec
	os.Exit(0)
	return RegisterFile{}
}

// 1NNN: JMP
func Jump2Addr(regFile RegisterFile, args OperationArgs) RegisterFile {
	return RegisterFile{
		Registers: regFile.Registers,
		PC:        args.Address,
	}
}

// 3XKK: SKNE
func SkipNextInstrIf(regFile RegisterFile, args OperationArgs) RegisterFile {
	// Skip next instr if V[X] == KK
	address := regFile.PC
	if regFile.Registers[args.XRegIndex] == byte(args.Value) {
		address += 4
	}

	return RegisterFile{
		Registers: regFile.Registers,
		PC:        address,
	}
}

// 6XKK: VX = KK
func AsignToReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	// V[X] = KK
	registers := regFile.Registers
	registers[args.XRegIndex] = byte(args.Value)

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 7XKK: VX = VX + KK
func AddValueToReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	var flags uint8
	// Overflow
	if uint32(registers[args.XRegIndex])+uint32(args.Value) > 0xF {
		flags = 0x1
	}
	registers[args.XRegIndex] += byte(args.Value) + registers[0xF]
	registers[0xF] = flags

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY0 VX = VY
func AsignReg2Reg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	registers[args.XRegIndex] = registers[args.YRegIndex]

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY1 VX = VX OR VY
func RegOrReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	registers[args.XRegIndex] |= registers[args.YRegIndex]

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY2 VX = VX AND VY
func RegAndReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	registers[args.XRegIndex] &= registers[args.YRegIndex]

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY3 VX = VX XOR VY
func RegXorReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	registers[args.XRegIndex] ^= registers[args.YRegIndex]

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY4 VX = VX + VY
func RegPlusReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	var flags uint8
	// Overflow
	if uint32(registers[args.XRegIndex])+uint32(registers[args.YRegIndex]) > 0xF {
		flags = 0x1
	}
	registers[args.XRegIndex] += registers[args.YRegIndex] + registers[0xF]
	registers[0xF] = flags

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY5 VX = VX - VY
func RegMinusReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	var Xvalue uint16 = uint16(registers[args.XRegIndex])
	// Underflow
	if registers[args.XRegIndex] < registers[args.YRegIndex] {
		registers[0xF] = 0x0
		Xvalue += 0x1 << 8
	} else {
		registers[0xF] = 0x1
	}

	registers[args.XRegIndex] = uint8(Xvalue - uint16(registers[args.YRegIndex]))

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8X06 VX = VX SHIFT RIGHT 1 (VX=VX/2)
func RegRShift(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	// VF = VX & 0x1
	registers[0xF] = registers[args.XRegIndex] & 0x1
	registers[args.XRegIndex] >>= 1

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8XY7 VX = VY - VX
func RegMinusRegInv(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	// Underflow
	if registers[args.YRegIndex] < registers[args.XRegIndex] {
		registers[0xF] = 1
	}
	registers[args.XRegIndex] = uint8(uint16(registers[args.YRegIndex]) - uint16(registers[args.XRegIndex]))

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// 8X0E VX = VX SHIFT LEFT 1 (VX=VX*2)
func RegLShift(regFile RegisterFile, args OperationArgs) RegisterFile {
	registers := regFile.Registers
	// VF = VX & 0x80
	registers[0xF] = registers[args.XRegIndex] & (0x1 << 7)

	registers[args.XRegIndex] <<= 1

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

// CXKK: VX = rand & KK
func RandomNumAndReg(regFile RegisterFile, args OperationArgs) RegisterFile {
	rNum := uint8(randSource.Int63() % 256)
	registers := regFile.Registers
	registers[args.XRegIndex] = rNum & uint8(args.Value)

	return RegisterFile{
		Registers: registers,
		PC:        regFile.PC + 2,
	}
}

func Run() {
	buffer := make([]byte, 2048)
	var opcode Opcode

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	// Read
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	i := 0
	regFile := NewRegisterFile()
	// Decode
	// for i := 0; i < bytesRead; i += 2 {
	for {
		opcode.Value = buffer[regFile.PC : regFile.PC+2]
		// opcode.Value = buffer[i : i+2]
		fmt.Println(opcode)

		fmt.Printf("i: %v, regFile: %s\n", i, regFile)
		// fmt.Printf("i: %v, regFile: %s\n", i, regFileAsBin(regFile))
		operation, args := Decode(opcode)
		// fmt.Printf("(op, args): (%s,%v)\n", getFunctionName(operation), args)
		// Execute

		regFile = operation(regFile, *args)
		i++
	}

}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <program>\n", os.Args[0])
		os.Exit(1)
	}

	Run()
}
