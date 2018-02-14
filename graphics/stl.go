package graphics

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/tidwall/gjson"
)

type STLWriter struct {
	Path string
}

func (stl *STLWriter) WriteModels(geoms []*bungie.DestinyGeometry) error {

	geom := geoms[0]
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

			data := geom.GetFileByName(vertexBuffers["file_name"].String()).Data
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
		indexBufferBytes := geom.GetFileByName(mesh["index_buffer"].Get("file_name").String()).Data

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
			f, err := os.OpenFile(stl.Path, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			bufferedWriter := bufio.NewWriter(f)
			bufferedWriter.Write([]byte(fmt.Sprintf("solid %s\n", meshName)))

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
							v[l] = positions[indexBuffer[start+j+k]][l]
						}

						bufferedWriter.Write([]byte(fmt.Sprintf("    vertex %.9f %.9f %.9f\n", v[0], v[1], v[2])))
					}
				} else {
					// write the three vertices to the file in forward order
					for k := 0; k < 3; k++ {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = positions[indexBuffer[start+j+k]][l]
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
