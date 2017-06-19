package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/tidwall/gjson"
)

const (
	BungieUrlPrefix      = "http://www.bungie.net"
	BungieGeometryPrefix = "/common/destiny_content/geometry/platform/mobile/geometry/"
	OffsetConstant       = 0.0
	ScaleConstant        = 1000.0
)

var (
	BungieApiKey       = os.Getenv("BUNGIE_API_KEY")
	LastWordGeometries = [5]string{"8458a82dec5290cdbc18fa568b94ff99.tgxm", "5bb9e8681f0423e7d89a1febe42457ec.tgxm", "cf97cbfcaae5736094c320b9e3378aa2.tgxm", "f878c2e86541fbf165747362eb3d54fc.tgxm", "4a00ec1e50813252fb0b1341adf1b675.tgxm"}
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

type DAEWriter struct {
}

type STLWriter struct {
}

func main() {

	for _, geometryFile := range LastWordGeometries {
		/*client := http.Client{}
		req, _ := http.NewRequest("GET", BungieUrlPrefix+BungieGeometryPrefix+geometryFile, nil)
		req.Header.Set("X-API-Key", BungieApiKey)
		response, _ := client.Do(req)

		bodyBytes, _ := ioutil.ReadAll(response.Body)
		ioutil.WriteFile("./local_tools/geom/"+geometryFile, bodyBytes, 0644)*/

		geometry := parseGeometryFile("./local_tools/geom/" + geometryFile)

		stlWriter := &STLWriter{}
		err := stlWriter.writeModel(geometry)
		if err != nil {
			fmt.Println("Error trying to write the model file!!: ", err.Error())
			return
		}
	}
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

	ioutil.WriteFile("./local_tools/lastword-meshes.json", geom.MeshesBytes, 0644)

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
		} else {
			fmt.Println("Found unknown vertex type!!")
		}
	}
	return result
}

func (stl *STLWriter) writeModel(geom *DestinyGeometry) error {

	result := gjson.Parse(string(geom.MeshesBytes))

	meshes := result.Get("render_model.render_meshes")
	if meshes.Exists() == false {
		err := errors.New("Error unmarshaling mesh JSON: render meshes not found")
		return err
	}

	fmt.Printf("Successfully parsed meshes JSON\n")

	for meshIndex, meshInterface := range meshes.Array() {
		mesh := meshInterface.Map()
		positions := [][]float64{}
		normals := [][]float64{}
		defVB := mesh["stage_part_vertex_stream_layout_definitions"].Array()[0].Map()["formats"].Array()

		for index, vbInterface := range mesh["vertex_buffers"].Array() {
			currentDefVB := defVB[index].Map()
			vertexBuffers := vbInterface.Map()
			stride := currentDefVB["stride"].Float()
			if stride != vertexBuffers["stride_byte_size"].Float() {
				return errors.New("Mismatched stride sizes found")
			}

			data := geom.getFileByName(vertexBuffers["file_name"].String()).Data
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
				case "_tfx_vb_semantic_normal":
					normals = parseVertex(data, elementType, int(elementOffset), int(stride))
				}
			}
		}

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

			primitiveType := int(part["primitive_type"].Float())
			increment := 1

			if primitiveType == 3 {
				// Process indexBuffer in sets of 3
				increment = 3
			} else if primitiveType == 5 {
				// Process indexBuffer as triangle strip
				increment = 1
				count -= 2
			} else {
				fmt.Println("Unknown primitive type, skipping this part...")
				continue
			}

			// We need to reverse the order of vertices every other iteration
			flip := false

			// Construct and write this mesh header
			meshName := fmt.Sprintf("%s_%d_%d", geom.Name, meshIndex, i)

			// TODO: This should check if the file exists and remove it first probably
			f, err := os.OpenFile("lastword.stl", os.O_WRONLY|os.O_APPEND, 0755)
			if err != nil {
				return err
			}
			defer f.Close()

			bufferedWriter := bufio.NewWriter(f)
			bufferedWriter.Write([]byte(fmt.Sprintf("solid %s\n", meshName)))

			fmt.Printf("Starting at index=%d, count=%d\n", start, count)
			for j := 0; j < count; j += increment {

				if (start + j + 2) >= len(indexBuffer) {
					fmt.Println("Skipping j=", j)
					continue
				}

				// Skip if any two of the indexBuffer match (ignoring lines or points)
				if indexBuffer[start+j+0] == indexBuffer[start+j+1] || indexBuffer[start+j+0] == indexBuffer[start+j+2] || indexBuffer[start+j+1] == indexBuffer[start+j+2] {
					flip = !flip
					continue
				}

				// Write the normal and loop start to file
				// the normal doesn't matter for this, the order of vertices does
				bufferedWriter.Write([]byte("facet normal 0.0 0.0 0.0\n  outer loop\n"))

				// flip the triangle only when using primitive_type 5
				if flip && (primitiveType == 5) {
					for k := 2; k >= 0; k-- {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						bufferedWriter.Write([]byte(fmt.Sprintf("    vertex %.9f %.9f %.9f\n", v[0], v[1], v[2])))
					}
				} else {
					// write the three vertices to the file in forward order
					for k := 0; k < 3; k++ {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						bufferedWriter.Write([]byte(fmt.Sprintf("    vertex %.9f %.9f %.9f\n", v[0], v[1], v[2])))
					}
				}

				// Write the loop and normal end to file
				bufferedWriter.Write([]byte("  endloop\nendfacet\n"))

				flip = !flip
			}

			bufferedWriter.Flush()
		}
	}

	return nil
}
