package vdf_go

import (
	"sync/atomic"
)

// VDF is the struct holding necessary state for a hash chain delay function.
type VDF struct {
	difficulty int
	input      [32]byte
	output     [516]byte
	outputChan chan [516]byte
	finished   int32
}

// size of long integers in quadratic function group
const sizeInBits = 2048

// New create a new instance of VDF.
func New(difficulty int, input [32]byte) *VDF {
	return &VDF{
		difficulty: difficulty,
		input:      input,
		outputChan: make(chan [516]byte),
	}
}

// GetOutputChannel returns the vdf output channel.
// VDF output consists of 258 bytes of serialized Y and  258 bytes of serialized Proof
func (vdf *VDF) GetOutputChannel() chan [516]byte {
	return vdf.outputChan
}

// Execute runs the VDF until it's finished and put the result into output channel.
// currently on i7-6700K, it takes about 14 seconds when iteration is set to 10000
func (vdf *VDF) Execute(stop <-chan struct{}) {
	atomic.StoreInt32(&vdf.finished, 0)

	yBuf, proofBuf := GenerateVDFWithStopChan(vdf.input[:], vdf.difficulty, sizeInBits, stop)

	copy(vdf.output[:], yBuf)
	copy(vdf.output[258:], proofBuf)

	go func() {
		vdf.outputChan <- vdf.output
	}()

	atomic.StoreInt32(&vdf.finished, 1)
}

// Verify runs the verification of generated proof
// currently on i7-6700K, verification takes about 350 ms
func (vdf *VDF) Verify(proof [516]byte) bool {
	return VerifyVDF(vdf.input[:], proof[:], vdf.difficulty, sizeInBits)
}

// IsFinished returns whether the vdf execution is finished or not.
func (vdf *VDF) IsFinished() bool {
	return atomic.LoadInt32(&vdf.finished) == 1
}

// GetOutput returns the vdf output, which can be bytes of 0s is the vdf is not finished.
func (vdf *VDF) GetOutput() [516]byte {
	return vdf.output
}
