package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

/****** TODO *******
- Request and parse asset manifest from the manifest endpoint
- Find a mapping of the item definition to the asset definition
- pull geom (.tgx) and texture (.tgx.bin) files out of the item definition for the male index set
- use those values in the the "SunshotGeometries" instead of the hardcoded .tgxm paths
*******************/

/**
	"mobileGearCDN": {
	"Geometry": "/common/destiny2_content/geometry/platform/mobile/geometry",
	"Texture": "/common/destiny2_content/geometry/platform/mobile/textures",
	"PlateRegion": "/common/destiny2_content/geometry/platform/mobile/plated_textures",
	"Gear": "/common/destiny2_content/geometry/gear",
	"Shader": "/common/destiny2_content/geometry/platform/mobile/shaders"
**/

const (
	BungieUrlPrefix          = "http://www.bungie.net"
	BungieGeometryPrefix     = "/common/destiny2_content/geometry/platform/mobile/geometry/"
	BungieTexturePrefix      = "/common/destiny2_content/geometry/platform/mobile/textures"
	BungieGearPrefix         = "/common/destiny2_content/geometry/gear"
	BungiePlatedRegionPrefix = "/common/destiny2_content/geometry/platform/mobile/plated_textures"
	BungieShaderPrefix       = "/common/destiny2_content/geometry/platform/mobile/shaders"
	OffsetConstant           = 0.0
	ScaleConstant            = 1000.0
	ModelNamePrefix          = "sunshot"
)

var (
	BungieApiKey = os.Getenv("BUNGIE_API_KEY")
	//LastWordGeometries = [5]string{"8458a82dec5290cdbc18fa568b94ff99.tgxm", "5bb9e8681f0423e7d89a1febe42457ec.tgxm", "cf97cbfcaae5736094c320b9e3378aa2.tgxm", "f878c2e86541fbf165747362eb3d54fc.tgxm", "4a00ec1e50813252fb0b1341adf1b675.tgxm"}
	//LastWordGeometries = [5]string{"8458a82dec5290cdbc18fa568b94ff99.tgxm", "5bb9e8681f0423e7d89a1febe42457ec.tgxm", "cf97cbfcaae5736094c320b9e3378aa2.tgxm"}
	//LastWordGeometries = [1]string{"5bb9e8681f0423e7d89a1febe42457ec.tgxm"}
	SunshotGeometries = []string{"21b966d2b3e9338b49b5243ecbdcccca.tgxm", "57152585e9f5300a5475478c8ea1f448.tgxm", "60fc8d77e90b35adddf0a99a99facf35.tgxm", "ae800a88325ed4b9e7bc32c86182ae75.tgxm", "defdbb6dcbce8fcd85422f44e53bc4c2.tgxm"}

	// Unused coord pair: [1.3330078125, 2.666015625]
	/*"texcoord_offset": [
	    0.401725,
	    0.400094
	  ],
	  "texcoord_scale": [
	    0.396719,
	    0.396719
	  ],*/
	//UnusedX        = 1.3330078125
	//UnusedY        = 2.66015625
	//UnusedX          = 1.333333333333333
	//UnusedY          = 2.666666666666667
	ManualOffsets  = [2]float32{-0.0, -0.0}
	TexcoordOffset = [2]float32{0.401725, 0.400094}
	TexcoordScale  = [2]float32{0.396719, 0.396719}
)

type DestinyGeometry struct {
	Extension   string
	HeaderSize  int32
	FileCount   int32
	Name        string
	MeshesBytes []byte
	Files       []*GeometryFile
}

type GeometryFile struct {
	Name      string
	StartAddr int64
	Length    int64
	Data      []byte
}

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

type GearAssetDefinition struct {
	Gear    []string       `json:"gear"`
	Content []*GearContent `json:"content"`
}

type GearContent struct {
	Platform        string                 `json:"platform"`
	Geometry        []string               `json:"geometry"`
	Textures        []string               `json:"textures"`
	DyeIndexSet     *IndexSet              `json:"dye_index_set"`
	RegionIndexSets map[string][]*IndexSet `json:"region_index_sets"`
}

func (g *GearContent) String() string {
	return fmt.Sprintf("%+v", *g)
}

type IndexSet struct {
	Textures []int `json:"textures"`
	Geometry []int `json:"geometry"`
}

func (is *IndexSet) String() string {
	return fmt.Sprintf("%+v", *is)
}

func main() {

	// TODO: Given some item hash, this should make a request for the item spec including
	// all of the geometry files and texture files

	if len(os.Args) < 2 {
		fmt.Println("Forgot to provide an item hash!")
		return
	}

	hash, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		fmt.Printf("Error parsing item hash from command line arg: %s\n", err.Error())
		return
	}
	db, err := GetAssetDBConnection()
	if err != nil {
		fmt.Printf("Error opening DB connection: %s\n", err.Error())
		return
	}

	definition, err := db.GetAssetDefinition(uint(hash))
	if err != nil {
		fmt.Printf("Failed to request item by hash from Asset DB: %s\n", err.Error())
		return
	}

	assetDefinition := &GearAssetDefinition{}
	decoder := json.NewDecoder(strings.NewReader(definition))
	err = decoder.Decode(assetDefinition)
	if err != nil {
		fmt.Printf("Failed to decode asset definition from the DB: %s\n", err.Error())
		return
	}

	fmt.Printf("Ready to go with this item def: %+v\n", assetDefinition)

	/***** TODO: Uncomment the stuff below this *****/

	// for index, geometryFile := range SunshotGeometries {
	// 	fmt.Println("Parsing geometry file... ", geometryFile)

	// 	/*client := http.Client{}
	// 	req, _ := http.NewRequest("GET", BungieUrlPrefix+BungieGeometryPrefix+geometryFile, nil)
	// 	req.Header.Set("X-API-Key", BungieApiKey)
	// 	response, _ := client.Do(req)

	// 	bodyBytes, _ := ioutil.ReadAll(response.Body)
	// 	ioutil.WriteFile("./local_tools/geom/"+geometryFile, bodyBytes, 0644)*/

	// 	geometry := parseGeometryFile("./local_tools/geom/geometry/" + geometryFile)

	// 	// stlWriter := &STLWriter{}
	// 	// err := stlWriter.writeModel(geometry)

	// 	daeWriter := &DAEWriter{fmt.Sprintf(ModelNamePrefix+"%d-auto.dae", index)}
	// 	err := daeWriter.writeModel(geometry)
	// 	if err != nil {
	// 		fmt.Println("Error trying to write the model file!!: ", err.Error())
	// 		return
	// 	}
	// }
}

func parseGeometryFile(path string) *DestinyGeometry {

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Failed to open geometry file with error: ", err.Error())
		return nil
	}
	defer f.Close()

	geom := &DestinyGeometry{}
	geom.Files = make([]*GeometryFile, 0)

	// Read file metadata
	buf := make([]byte, 272)
	f.Read(buf)
	metaBuffer := bytes.NewBuffer(buf)

	extension := make([]byte, 4)
	binary.Read(metaBuffer, binary.LittleEndian, extension)

	geom.Extension = string(extension)

	// Unknown
	metaBuffer.Next(4)

	binary.Read(metaBuffer, binary.LittleEndian, &geom.HeaderSize)
	binary.Read(metaBuffer, binary.LittleEndian, &geom.FileCount)
	nameBuf := make([]byte, 256)
	binary.Read(metaBuffer, binary.LittleEndian, &nameBuf)

	endOfName := bytes.IndexByte(nameBuf, 0)
	geom.Name = string(nameBuf[:endOfName])

	// Read each of the individual files
	for i := int32(0); i < geom.FileCount; i++ {
		file := &GeometryFile{}

		nameBuf := make([]byte, 256)
		f.Read(nameBuf)
		n := bytes.IndexByte(nameBuf, 0)

		file.Name = string(nameBuf[:n])

		startAddrBuf := make([]byte, 8)
		f.Read(startAddrBuf)
		binary.Read(bytes.NewBuffer(startAddrBuf), binary.LittleEndian, &file.StartAddr)

		lengthBuffer := make([]byte, 8)
		f.Read(lengthBuffer)
		binary.Read(bytes.NewBuffer(lengthBuffer), binary.LittleEndian, &file.Length)

		geom.Files = append(geom.Files, file)

		fmt.Printf("Finished reading file metadata: %+v\n", file)
	}

	for _, file := range geom.Files {
		f.Seek(file.StartAddr, 0)

		file.Data = make([]byte, file.Length)
		f.Read(file.Data)

		if file.Name == "render_metadata.js" {
			fmt.Println("Found render_metadata.js file!!")
			geom.MeshesBytes = file.Data
		}
	}

	ioutil.WriteFile("./local_tools/"+ModelNamePrefix+"-meshes.json", geom.MeshesBytes, 0644)

	return geom
}

func (geom *DestinyGeometry) getFileByName(name string) *GeometryFile {

	for _, file := range geom.Files {
		if file.Name == name {
			return file
		}
	}
	return nil
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

func (dae *DAEWriter) writeModel(geom *DestinyGeometry) error {

	result := gjson.Parse(string(geom.MeshesBytes))

	meshes := result.Get("render_model.render_meshes")
	if meshes.Exists() == false {
		err := errors.New("Error unmarshaling mesh JSON: render meshes not found")
		return err
	}

	fmt.Printf("Successfully parsed meshes JSON\n")

	positionVertices := make([]float64, 0, 1024)
	texcoords := make([]float32, 0, 1024)
	for meshIndex, meshInterface := range meshes.Array() {
		if meshIndex != 1 {
			//continue
		}

		mesh := meshInterface.Map()
		positions := [][]float64{}
		normals := [][]float64{}
		innerTexcoords := [][]float32{}
		adjustments := [][]float32{}
		defVB := mesh["stage_part_vertex_stream_layout_definitions"].Array()[0].Map()["formats"].Array()

		for index, vbInterface := range mesh["vertex_buffers"].Array() {
			currentDefVB := defVB[index].Map()
			vertexBuffers := vbInterface.Map()
			stride := currentDefVB["stride"].Float()
			if stride != vertexBuffers["stride_byte_size"].Float() {
				return errors.New("Mismatched stride sizes found")
			}

			data := geom.getFileByName(vertexBuffers["file_name"].String()).Data
			fmt.Println("Reading data from file: ", vertexBuffers["file_name"].String())
			if data == nil {
				return errors.New("Missing geometry file by name: " + vertexBuffers["file_name"].String())
			}

			for _, elementInterface := range currentDefVB["elements"].Array() {
				element := elementInterface.Map()
				elementType := element["type"].String()
				elementOffset := element["offset"].Float()

				switch element["semantic"].String() {
				case "_tfx_vb_semantic_position":
					positions = parseVertex(data, elementType, int(elementOffset), int(stride))
					fmt.Printf("Found positions: %d\n", len(positions))
				case "_tfx_vb_semantic_normal":
					normals = parseVertex(data, elementType, int(elementOffset), int(stride))
					fmt.Printf("Found normals: len=%d\n", len(normals))
				case "_tfx_vb_semantic_texcoord":
					if elementType != "_vertex_format_attribute_float2" {
						innerTexcoords = parseVertex32(data, elementType, int(elementOffset), int(stride))
						fmt.Printf("Found textcoords: len=%d\n", len(innerTexcoords))
					} else {
						adjustments = parseVertex32(data, elementType, int(elementOffset), int(stride))
						fmt.Printf("Found adjustments: %d\n", len(adjustments))
						//fmt.Printf("Found float texcoords: %+v\n", throwaway)
					}
				}
			}
		}

		minS := [2]float32{0.0, 0.0}
		minT := [2]float32{0.0, 0.0}
		maxS := [2]float32{0.0, 0.0}
		maxT := [2]float32{0.0, 0.0}

		if len(positions) == 0 || len(normals) == 0 || len(positions) != len(normals) {
			return errors.New("Positions slice is not the same size as the normals slice")
		}

		// Parse the index buffer
		indexBuffer := make([]int16, 0)
		indexBufferBytes := geom.getFileByName(mesh["index_buffer"].Get("file_name").String()).Data

		for i := 0; i < len(indexBufferBytes); i += 2 {

			var index int16
			binary.Read(bytes.NewBuffer(indexBufferBytes[i:i+2]), binary.LittleEndian, &index)
			indexBuffer = append(indexBuffer, index)
		}

		parts := mesh["stage_part_list"].Array()

		// Loop through all the parts in the mesh
		for i, partInterface := range parts {
			part := partInterface.Map()
			start := int(part["start_index"].Float())
			count := int(part["index_count"].Float())

			// Check if this part has been duplciated
			ignore := false
			for j := 0; j < i; j++ {
				jStartIndex := parts[j].Map()["start_index"].Float()
				jIndexCount := parts[j].Map()["index_count"].Float()
				if (start == int(jStartIndex)) || (count == int(jIndexCount)) {
					ignore = true
					break
				}
			}

			lodCategoryValue := int(part["lod_category"].Get("value").Float())
			if (ignore == true) || (lodCategoryValue > 1) {
				continue
			}

			// PrimitiveType, 3=TRIANGLES, 5=TRIANGLE_STRIP
			// https://stackoverflow.com/questions/3485034/convert-triangle-strips-to-triangles

			// Process indexBuffer in sets of 3
			primitiveType := int(part["primitive_type"].Float())
			increment := 3

			if primitiveType == 5 {
				// Process indexBuffer as triangle strip
				increment = 1
				count -= 2
			} else if primitiveType != 3 {
				fmt.Println("Unknown primitive type, skipping this part...")
				continue
			}

			// Construct and write this mesh header
			for j := 0; j < count; j += increment {

				// Skip if any two of the indexBuffer match (ignoring lines or points)
				if indexBuffer[start+j+0] == indexBuffer[start+j+1] || indexBuffer[start+j+0] == indexBuffer[start+j+2] || indexBuffer[start+j+1] == indexBuffer[start+j+2] {
					continue
				}

				tri := [3]int{0, 1, 2}
				if (primitiveType == 3) || ((j & 1) == 1) {
					tri = [3]int{2, 1, 0}
				}

				for k := 0; k < 3; k++ {
					v := [4]float64{}
					for l := 0; l < 4; l++ {
						v[l] = (positions[indexBuffer[start+j+tri[k]]][l] + OffsetConstant) * ScaleConstant
					}

					tex := [2]float32{}
					//fmt.Printf("Normal indices: ")
					for l := 0; l < 2; l++ {
						offset := TexcoordOffset[l]
						scale := TexcoordScale[l]
						adjustment := float32(1.0)
						if len(adjustments) > 0 {
							adjustment = adjustments[indexBuffer[start+j+tri[k]]][l]
						}
						//adjustment = 1.0
						//fmt.Printf("%d,", indexBuffer[start+j+tri[k]])
						tex[l] = (((texcoordVal(innerTexcoords[indexBuffer[start+j+tri[k]]][l])).normalize(16) * scale * adjustment) + offset) + ManualOffsets[l]
					}

					if tex[0] < minS[0] {
						minS = tex
					}
					if tex[1] < minT[1] {
						minT = tex
					}
					if tex[0] > maxS[0] {
						maxS = tex
					}
					if tex[1] > maxT[1] {
						maxT = tex
					}

					//fmt.Println()
					positionVertices = append(positionVertices, v[0], v[1], v[2])
					texcoords = append(texcoords, tex[0], tex[1])
				}
			}
		}

		fmt.Printf("minS=%v, maxS=%v, minT=%v, maxT=%v\n", minS, maxS, minT, maxT)

		// for _, buffer := range positions {
		// 	positionVertices = append(positionVertices, buffer[0], buffer[1], buffer[2])
		// }
	}

	dae.writeXML(positionVertices, texcoords)

	return nil
}
