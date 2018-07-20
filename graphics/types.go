package graphics

import (
	"image/draw"
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

// PBRTextureCollection will contain the diffrent textures that need to be used for physically based
// rendering (PBR).
type PBRTextureCollection struct {
	Metalness, Roughness, AmbientOcclusion, Emissive draw.Image
	ImageFormat                                      string
}

type texturePlate struct {
	name string
	tag  string
	size [2]int
	data draw.Image

	// IDs and other names written to the resulting COLLADA (.dae) file for referencing in other
	// spots in the file
	libraryImagesID   string
	libraryEffectsID  string
	libraryMaterialID string
}

type processedOutput struct {
	// One slice of floats for each mesh (parts are all included in a single slice)
	// first index is the geometry index, second is the position, normal, texcoord for that texture.
	positionVertices [][]float64
	normalValues     [][]float64
	texcoords        [][]float32

	// These indices map a texture plate index for a geometry to
	// an entry in the texturePlates array
	plateIndices           []int
	texturePlates          [10]*texturePlate
	normalTexturePlates    [10]*texturePlate
	gearstackTexturePlates [10]*texturePlate
	pbrTextures            [10]*PBRTextureCollection
}
