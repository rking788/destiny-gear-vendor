package graphics

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	includeTextures = true
)

type texcoordVal float32

func (f texcoordVal) normalize(bitNum float32) float32 {
	max := texcoordVal(math.Pow(2, float64(bitNum-1)) - 1.0)
	ret := math.Max(float64(f/max), -1)
	return float32(ret)
}

func (f texcoordVal) unsignedNormalize(bitNum float64) float32 {
	max := texcoordVal(math.Pow(2, bitNum) - 1)
	return float32(float64(f) / float64(max))
}

func parseVertex(data []byte, vertexType string, offset int, stride int) [][]float64 {

	result := make([][]float64, 0, 10)
	for i := offset; i < len(data); i += stride {

		if vertexType == "_vertex_format_attribute_short4" {
			out := make([]int16, 4)
			binary.Read(bytes.NewBuffer(data[i:i+8]), binary.LittleEndian, out)
			outFloats := []float64{float64(out[0]), float64(out[1]), float64(out[2]), float64(out[3])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_float4" {
			out := make([]float32, 4)
			binary.Read(bytes.NewBuffer(data[i:i+16]), binary.LittleEndian, out)
			outFloats := []float64{float64(out[0]), float64(out[1]), float64(out[2]), float64(out[3])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_short2" {
			out := make([]int16, 2)
			binary.Read(bytes.NewBuffer(data[i:i+4]), binary.LittleEndian, out)
			outFloats := []float64{float64(out[0]), float64(out[1])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_float2" {
			out := make([]float32, 2)
			binary.Read(bytes.NewBuffer(data[i:i+8]), binary.LittleEndian, out)
			outFloats := []float64{float64(out[0]), float64(out[1])}
			result = append(result, outFloats)
		} else {
			fmt.Println("Found unknown vertex type!!")
		}
	}
	return result
}

func parseVertex32(data []byte, vertexType string, offset int, stride int) [][]float32 {

	result := make([][]float32, 0, 10)
	for i := offset; i < len(data); i += stride {

		if vertexType == "_vertex_format_attribute_short2" {
			out := make([]int16, 2)
			binary.Read(bytes.NewBuffer(data[i:i+4]), binary.LittleEndian, out)
			outFloats := []float32{float32(out[0]), float32(out[1])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_float2" {
			out := make([]float32, 2)
			binary.Read(bytes.NewBuffer(data[i:i+8]), binary.LittleEndian, out)
			outFloats := []float32{float32(out[0]), float32(out[1])}
			result = append(result, outFloats)
		} else {
			fmt.Println("Found unknown vertex type!!")
		}
	}
	return result
}
