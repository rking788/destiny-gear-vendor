package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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
			//fmt.Printf("Parsing bytes: [% x]\n", data[i:i+8])
			binary.Read(bytes.NewBuffer(data[i:i+8]), binary.LittleEndian, out)
			//fmt.Printf("binary.Read shorts: %+v\n", out)
			outFloats := []float64{float64(out[0]), float64(out[1]), float64(out[2]), float64(out[3])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_float4" {
			out := make([]float32, 4)
			//fmt.Printf("Parsing bytes: [% x]\n", data[i:i+16])
			binary.Read(bytes.NewBuffer(data[i:i+16]), binary.LittleEndian, out)
			//fmt.Printf("binary.Read floats: %+v\n", out)
			outFloats := []float64{float64(out[0]), float64(out[1]), float64(out[2]), float64(out[3])}
			result = append(result, outFloats)
		} else {
			fmt.Println("Found unknown vertex type!!")
		}
	}
	return result
}

func (stl *STLWriter) writeModel(geom *DestinyGeometry) error {
	var result map[string]interface{}

	err := json.Unmarshal(geom.MeshesBytes, &result)
	if err != nil {
		fmt.Println("Error unmarshaling mesh JSON: ", err.Error())
		return err
	}

	meshes := result["render_model"].(map[string]interface{})["render_meshes"].([]interface{})

	fmt.Printf("Successfully parsed meshes JSON\n")

	for meshIndex, meshInterface := range meshes {
		mesh := meshInterface.(map[string]interface{})
		positions := [][]float64{}
		normals := [][]float64{}
		defVB := mesh["stage_part_vertex_stream_layout_definitions"].([]interface{})[0].(map[string]interface{})["formats"].([]interface{})

		for index, vbInterface := range mesh["vertex_buffers"].([]interface{}) {
			currentDefVB := defVB[index].(map[string]interface{})
			vertexBuffers := vbInterface.(map[string]interface{})
			stride := currentDefVB["stride"].(float64)
			if stride != vertexBuffers["stride_byte_size"].(float64) {
				return errors.New("Mismatched stride sizes found")
			}

			data := geom.getFileByName(vertexBuffers["file_name"].(string)).Data
			if data == nil {
				return errors.New("Missing geometry file by name: " + vertexBuffers["file_name"].(string))
			}

			for _, elementInterface := range currentDefVB["elements"].([]interface{}) {
				element := elementInterface.(map[string]interface{})
				elementType := element["type"].(string)
				elementOffset := element["offset"].(float64)

				switch element["semantic"].(string) {
				case "_tfx_vb_semantic_position":
					positions = parseVertex(data, elementType, int(elementOffset), int(stride))
				case "_tfx_vb_semantic_normal":
					normals = parseVertex(data, elementType, int(elementOffset), int(stride))
				}
			}
		}

		//fmt.Printf("Found positions: %v\n", positions)
		//fmt.Printf("Found normals: %v\n", normals)
		if len(positions) == 0 || len(normals) == 0 || len(positions) != len(normals) {
			return errors.New("Positions slice is not the same size as the normals slice")
		}

		// Parse the index buffer
		indexBuffer := make([]int16, 0)

		indexBufferBytes := geom.getFileByName(mesh["index_buffer"].(map[string]interface{})["file_name"].(string)).Data

		fmt.Println("indexbuffer length in bytes: ", len(indexBufferBytes))
		for i := 0; i < len(indexBufferBytes); i += 2 {

			var index int16
			binary.Read(bytes.NewBuffer(indexBufferBytes[i:i+2]), binary.LittleEndian, &index)
			indexBuffer = append(indexBuffer, index)
		}

		parts := mesh["stage_part_list"].([]interface{})

		// Loop through all the parts in the mesh
		for i, partInterface := range parts {
			part := partInterface.(map[string]interface{})
			start := int(part["start_index"].(float64))
			count := int(part["index_count"].(float64))

			// Check if this part has been duplciated
			ignore := false
			for j := 0; j < i; j++ {
				jStartIndex := parts[j].(map[string]interface{})["start_index"].(float64)
				jIndexCount := parts[j].(map[string]interface{})["index_count"].(float64)
				if (start == int(jStartIndex)) || (count == int(jIndexCount)) {
					ignore = true
					break
				}
			}

			lodCategoryValue := int(part["lod_category"].(map[string]interface{})["value"].(float64))
			if (ignore == true) || (lodCategoryValue > 1) {
				continue
			}

			primitiveType := int(part["primitive_type"].(float64))
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
				bufferedWriter.Write([]byte("facet normal 0.0 0.0 0.0\n"))
				bufferedWriter.Write([]byte("  outer loop\n"))

				// flip the triangle only when using primitive_type 5
				if flip && (primitiveType == 5) {
					for k := 2; k >= 0; k-- {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						bufferedWriter.Write([]byte(fmt.Sprintf("    vertex %f %f %f\n", v[0], v[1], v[2])))
					}
				} else {
					// write the three vertices to the file in forward order
					for k := 0; k < 3; k++ {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						bufferedWriter.Write([]byte(fmt.Sprintf("    vertex %f %f %f\n", v[0], v[1], v[2])))
					}
				}

				// Write the loop and normal end to file
				bufferedWriter.Write([]byte("  endloop\n"))
				bufferedWriter.Write([]byte("endfacet\n"))

				flip = !flip
			}

			bufferedWriter.Flush()
		}
	}

	return nil
}
