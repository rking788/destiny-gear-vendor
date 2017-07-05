package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"strings"

	"github.com/beevik/etree"
	"github.com/tidwall/gjson"
)

const (
	BungieUrlPrefix      = "http://www.bungie.net"
	BungieGeometryPrefix = "/common/destiny_content/geometry/platform/mobile/geometry/"
	OffsetConstant       = 0.0
	ScaleConstant        = 1000.0
)

var (
	BungieApiKey = os.Getenv("BUNGIE_API_KEY")
	//LastWordGeometries = [5]string{"8458a82dec5290cdbc18fa568b94ff99.tgxm", "5bb9e8681f0423e7d89a1febe42457ec.tgxm", "cf97cbfcaae5736094c320b9e3378aa2.tgxm", "f878c2e86541fbf165747362eb3d54fc.tgxm", "4a00ec1e50813252fb0b1341adf1b675.tgxm"}
	LastWordGeometries  = [1]string{"5bb9e8681f0423e7d89a1febe42457ec.tgxm"}
	NormalizationFactor = 65535.0
	TexcoordOffset      = [2]float64{0.401725, 0.400094}
	TexcoordScale       = [2]float64{0.396719, 0.396719}
	//TexcoordOffset = [2]float64{0.0, 0.0}
	//TexcoordScale  = [2]float64{1.0, 1.0}
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
	Path string
}

type STLWriter struct {
}

func main() {

	for index, geometryFile := range LastWordGeometries {
		fmt.Println("Parsing geometry file... ", geometryFile)

		/*client := http.Client{}
		req, _ := http.NewRequest("GET", BungieUrlPrefix+BungieGeometryPrefix+geometryFile, nil)
		req.Header.Set("X-API-Key", BungieApiKey)
		response, _ := client.Do(req)

		bodyBytes, _ := ioutil.ReadAll(response.Body)
		ioutil.WriteFile("./local_tools/geom/"+geometryFile, bodyBytes, 0644)*/

		geometry := parseGeometryFile("./local_tools/geom/geometry/" + geometryFile)

		// stlWriter := &STLWriter{}
		// err := stlWriter.writeModel(geometry)

		daeWriter := &DAEWriter{fmt.Sprintf("lastword%d-auto.dae", index)}
		err := daeWriter.writeModel(geometry)
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
		} else if vertexType == "_vertex_format_attribute_short2" {
			out := make([]uint16, 2)
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
			f, err := os.OpenFile("lastword.stl", os.O_RDWR|os.O_CREATE, 0644)
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

func (dae *DAEWriter) writeModel(geom *DestinyGeometry) error {

	result := gjson.Parse(string(geom.MeshesBytes))

	meshes := result.Get("render_model.render_meshes")
	if meshes.Exists() == false {
		err := errors.New("Error unmarshaling mesh JSON: render meshes not found")
		return err
	}

	fmt.Printf("Successfully parsed meshes JSON\n")

	positionVertices := make([]float64, 0, 1024)
	texcoords := make([]float64, 0, 1024)
	for meshIndex, meshInterface := range meshes.Array() {
		if meshIndex != 1 {
			//continue
		}

		mesh := meshInterface.Map()
		positions := [][]float64{}
		normals := [][]float64{}
		innerTexcoords := [][]float64{}
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
					fmt.Printf("Found positions: %d\n", len(positions))
				case "_tfx_vb_semantic_normal":
					normals = parseVertex(data, elementType, int(elementOffset), int(stride))
					fmt.Printf("Found normals: len=%d\n", len(normals))
				case "_tfx_vb_semantic_texcoord":
					if elementType != "_vertex_format_attribute_float2" {
						innerTexcoords = parseVertex(data, elementType, int(elementOffset), int(stride))
						fmt.Printf("Found textcoords: len=%d\n", len(innerTexcoords))
					} else {
						_ = parseVertex(data, elementType, int(elementOffset), int(stride))
						//fmt.Printf("Found float texcoords: %+v\n", throwaway)
					}
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

				// flip the triangle only when using primitive_type 5
				if flip && (primitiveType == 5) {
					for k := 2; k >= 0; k-- {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						tex := [2]float64{}
						for l := 0; l < 2; l++ {
							offset := TexcoordOffset[l]
							scale := TexcoordScale[l]
							tex[l] = (((innerTexcoords[indexBuffer[start+j+k]][l]) / NormalizationFactor) * scale) + offset
						}

						positionVertices = append(positionVertices, v[0], v[1], v[2])
						texcoords = append(texcoords, tex[0], tex[1])
					}
				} else {
					// write the three vertices to the file in forward order
					for k := 0; k < 3; k++ {
						v := [4]float64{}
						for l := 0; l < 4; l++ {
							v[l] = (positions[indexBuffer[start+j+k]][l] + OffsetConstant) * ScaleConstant
						}

						tex := [2]float64{}
						for l := 0; l < 2; l++ {
							offset := TexcoordOffset[l]
							scale := TexcoordScale[l]
							tex[l] = (((innerTexcoords[indexBuffer[start+j+k]][l]) / NormalizationFactor) * scale) + offset
						}

						positionVertices = append(positionVertices, v[0], v[1], v[2])
						texcoords = append(texcoords, tex[0], tex[1])
					}
				}

				flip = !flip
			}
		}

		// for _, coordPair := range innerTexcoords {
		// 	adjusted0 := (float64(coordPair[0])/32768.0 + OffsetConstant) * 1.0
		// 	adjusted1 := ((coordPair[1] / 32768.0) + OffsetConstant) * 1.0
		// 	texcoords = append(texcoords, adjusted0, adjusted1)
		// }
	}

	dae.writeXML(positionVertices, texcoords)

	return nil
}

func (dae *DAEWriter) writeXML(positions, texcoords []float64) error {

	if len(positions) <= 0 {
		return nil
	}

	posWriter := bytes.NewBufferString("")
	normalWriter := bytes.NewBufferString("")
	trianglesWriter := bytes.NewBufferString("")
	texcoordsWriter := bytes.NewBufferString("")

	for i, pos := range positions {
		posWriter.WriteString(fmt.Sprintf("%f ", pos))
		normalWriter.Write([]byte("0 "))
		trianglesWriter.WriteString(fmt.Sprintf("%d ", i))
	}

	for _, coord := range texcoords {
		texcoordsWriter.WriteString(fmt.Sprintf("%f ", coord))
	}

	doc, colladaRoot := NewColladaDoc()

	writeAssetElement(colladaRoot)
	writeImageLibraryElement(colladaRoot)

	// TODO: These cannot be empty, need to add solid material data
	materialID := "STL_material"
	materialEffectName := "effect_STL_material"
	writeLibraryMaterials(colladaRoot, materialID, materialEffectName)

	writeLibraryEffects(colladaRoot, materialEffectName)

	geometryID := "3054293897-0_0_1"
	writeLibraryGeometries(colladaRoot, posWriter, normalWriter, texcoordsWriter, trianglesWriter, len(positions), len(texcoords), geometryID)

	sceneID := 1
	sceneName := fmt.Sprintf("scene%d", sceneID)
	// TODO: This will be the same as the geometry names, should be taken from teh Geometry struct
	nodeName := geometryID

	writeLibraryVisualScenes(colladaRoot, sceneName, nodeName, geometryID)

	writeSceneElement(colladaRoot, sceneName)

	doc.Indent(2)
	//doc.WriteTo(os.Stdout)

	// Write this to a file now
	outF, err := os.OpenFile(dae.Path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer outF.Close()

	doc.WriteTo(outF)

	return nil
}

// NewColladaDoc will open a new XML document and write the correct header metadata and
// return the root XML element.
func NewColladaDoc() (*etree.Document, *etree.Element) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	colladaRoot := doc.CreateElement("COLLADA")
	colladaRoot.CreateAttr("xmlns", "http://www.collada.org/2005/11/COLLADASchema")
	colladaRoot.CreateAttr("version", "1.4.1")

	return doc, colladaRoot
}

func writeAssetElement(parent *etree.Element) {
	asset := parent.CreateElement("asset")
	asset.CreateElement("contributor").CreateElement("publishing_tool").CreateCharData("Destiny DAE Generator")

	timestamp := time.Now().UTC().Format(time.RFC3339)
	asset.CreateElement("created").CreateCharData(timestamp)
	asset.CreateElement("modified").CreateCharData(timestamp)
	asset.CreateElement("up_axis").CreateCharData("Y_UP")
}

func writeImageLibraryElement(parent *etree.Element) {

	imgName := "742477913_exotic_01_frame_gbit_384_192-2.jpg"
	libImages := parent.CreateElement("library_images")
	img1 := libImages.CreateElement("image")
	img1.CreateAttr("id", "image1")
	initFrom := img1.CreateElement("init_from")
	initFrom.CreateCharData(fmt.Sprintf("./%s", imgName))
}

func writeLibraryMaterials(parent *etree.Element, materialID, materialEffectName string) {
	libraryMaterials := parent.CreateElement("library_materials")

	material := libraryMaterials.CreateElement("material")
	material.CreateAttr("id", materialID)
	material.CreateAttr("name", materialID)
	material.CreateElement("instance_effect").CreateAttr("url", fmt.Sprintf("#%s", materialEffectName))

	texMaterial := libraryMaterials.CreateElement("material")
	texMaterial.CreateAttr("id", "lambert1")
	texMaterial.CreateAttr("name", "lambert1")
	texMaterial.CreateElement("instance_effect").CreateAttr("url", fmt.Sprintf("#effect_lambert1"))
}

func writeLibraryEffects(parent *etree.Element, materialEffectName string) {
	libraryEffects := parent.CreateElement("library_effects")

	effect := libraryEffects.CreateElement("effect")
	effect.CreateAttr("id", materialEffectName)

	profileCommon := effect.CreateElement("profile_COMMON")
	technique := profileCommon.CreateElement("technique")
	technique.CreateAttr("sid", "common")

	phong := technique.CreateElement("phong")
	ambient := phong.CreateElement("ambient")
	ambient.CreateElement("color").CreateCharData("0 0 0 1")

	diffuse := phong.CreateElement("diffuse")
	diffuse.CreateElement("color").CreateCharData("1 1 1 1")

	reflective := phong.CreateElement("reflective")
	reflective.CreateElement("color").CreateCharData("0 0 0 1")

	transparent := phong.CreateElement("transparent")
	transparent.CreateAttr("opaque", "A_ONE")
	transparent.CreateElement("color").CreateCharData("1 1 1 1")

	transparency := phong.CreateElement("transparency")
	transparency.CreateElement("float").CreateCharData("1")

	indexOfRefraction := phong.CreateElement("index_of_refraction")
	indexOfRefraction.CreateElement("float").CreateCharData("1")

	/**
	 * Lambert1 effects
	 **/
	lambertEffect := libraryEffects.CreateElement("effect")
	lambertEffect.CreateAttr("id", "effect_lambert1")

	lambertProfileCommon := lambertEffect.CreateElement("profile_COMMON")

	imgSurfNewParam := lambertProfileCommon.CreateElement("newparam")
	imgSurfNewParam.CreateAttr("sid", "ID2_image1_surface")

	surface := imgSurfNewParam.CreateElement("surface")
	surface.CreateAttr("type", "2D")
	surface.CreateElement("init_from").CreateCharData("image1")

	imageNewParam := lambertProfileCommon.CreateElement("newparam")
	imageNewParam.CreateAttr("sid", "ID2_image1")

	sampler2D := imageNewParam.CreateElement("sampler2D")
	sampler2D.CreateElement("source").CreateCharData("ID2_image1_surface")
	sampler2D.CreateElement("wrap_s").CreateCharData("CLAMP")
	sampler2D.CreateElement("wrap_t").CreateCharData("CLAMP")
	sampler2D.CreateElement("minfilter").CreateCharData("LINEAR")
	sampler2D.CreateElement("magfilter").CreateCharData("LINEAR")
	sampler2D.CreateElement("mipfilter").CreateCharData("LINEAR")

	lambertTechnique := lambertProfileCommon.CreateElement("technique")
	lambertTechnique.CreateAttr("sid", "common")

	blinn := lambertTechnique.CreateElement("blinn")
	blinnAmbient := blinn.CreateElement("ambient")
	blinnAmbient.CreateElement("color").CreateCharData("1 1 1 1")

	blinnDiffuse := blinn.CreateElement("diffuse")
	diffuseTexture := blinnDiffuse.CreateElement("texture")
	diffuseTexture.CreateAttr("texture", "ID2_image1")
	diffuseTexture.CreateAttr("texcoord", "CHANNEL2")

	blinnSpecular := blinn.CreateElement("specular")
	blinnSpecular.CreateElement("color").CreateCharData("0.496564 0.496564 0.496564 1")

	blinnShininess := blinn.CreateElement("shininess")
	blinnShininess.CreateElement("float").CreateCharData("0.022516")

	blinnReflective := blinn.CreateElement("reflective")
	blinnReflective.CreateElement("color").CreateCharData("0 0 0 1")

	blinnTransparent := blinn.CreateElement("transparent")
	blinnTransparent.CreateAttr("opaque", "A_ONE")
	blinnTransparent.CreateElement("color").CreateCharData("0.998203 1 1 1")

	blinnTransparency := blinn.CreateElement("transparency")
	blinnTransparency.CreateElement("float").CreateCharData("1")

	blinnIndexOfRefraction := blinn.CreateElement("index_of_refraction")
	blinnIndexOfRefraction.CreateElement("float").CreateCharData("1")

	lambertExtra := lambertEffect.CreateElement("extra")
	lambertExtraTech := lambertExtra.CreateElement("technique")
	lambertExtraTech.CreateElement("litPerPixel").CreateCharData("1")
	lambertExtraTech.CreateElement("ambient_diffuse_lock").CreateCharData("1")

	lambertExtraIntensities := lambertExtraTech.CreateElement("intensities")
	emission := lambertExtraIntensities.CreateElement("emission")
	emission.CreateElement("float").CreateCharData("0.5")
}

func writeLibraryGeometries(parent *etree.Element, posWriter, normalWriter, texcoordWriter, trianglesWriter *bytes.Buffer, positionCount int, texcoordCount int, geometryID string) {
	libGeometries := parent.CreateElement("library_geometries")

	// TODO: This should actually be read from the DestinyGeometry type
	geometry := libGeometries.CreateElement("geometry")
	geometry.CreateAttr("id", geometryID)
	geometry.CreateAttr("name", geometryID)

	mesh := geometry.CreateElement("mesh")

	posSource := mesh.CreateElement("source")
	posSource.CreateAttr("id", "geometrySource1")

	posFloatArray := posSource.CreateElement("float_array")
	posFloatArray.CreateAttr("id", "ID2-array")
	posFloatArray.CreateAttr("count", fmt.Sprintf("%d", positionCount))
	posFloatArray.CreateCharData(strings.TrimSpace(posWriter.String()))

	techniqueCommon := posSource.CreateElement("technique_common")
	accessor := techniqueCommon.CreateElement("accessor")
	accessor.CreateAttr("source", "#ID2-array")
	accessor.CreateAttr("count", fmt.Sprintf("%d", positionCount/3))
	accessor.CreateAttr("stride", "3")

	xParam := accessor.CreateElement("param")
	xParam.CreateAttr("name", "X")
	xParam.CreateAttr("type", "float")
	yParam := accessor.CreateElement("param")
	yParam.CreateAttr("name", "Y")
	yParam.CreateAttr("type", "float")
	zParam := accessor.CreateElement("param")
	zParam.CreateAttr("name", "Y")
	zParam.CreateAttr("type", "float")

	normalsSource := mesh.CreateElement("source")
	normalsSource.CreateAttr("id", "geometrySource2")

	normFloatArray := normalsSource.CreateElement("float_array")
	normFloatArray.CreateAttr("id", "ID4-array")
	normFloatArray.CreateAttr("count", fmt.Sprintf("%d", positionCount))
	normFloatArray.CreateCharData(strings.TrimSpace(normalWriter.String()))

	normTechniqueCommon := normalsSource.CreateElement("technique_common")
	normAccessor := normTechniqueCommon.CreateElement("accessor")
	normAccessor.CreateAttr("source", "#ID4-array")
	normAccessor.CreateAttr("count", fmt.Sprintf("%d", positionCount/3))
	normAccessor.CreateAttr("stride", "3")

	normXParam := normAccessor.CreateElement("param")
	normXParam.CreateAttr("name", "X")
	normXParam.CreateAttr("type", "float")
	normYParam := normAccessor.CreateElement("param")
	normYParam.CreateAttr("name", "Y")
	normYParam.CreateAttr("type", "float")
	normZParam := normAccessor.CreateElement("param")
	normZParam.CreateAttr("name", "Y")
	normZParam.CreateAttr("type", "float")

	texcoordSource := mesh.CreateElement("source")
	texcoordSource.CreateAttr("id", "geometrySource7")

	texcoordFloatArray := texcoordSource.CreateElement("float_array")
	texcoordFloatArray.CreateAttr("id", "ID8-array")
	texcoordFloatArray.CreateAttr("count", fmt.Sprintf("%d", texcoordCount))
	texcoordFloatArray.CreateCharData(strings.TrimSpace(texcoordWriter.String()))

	texcoordTechCommon := texcoordSource.CreateElement("technique_common")
	texcoordAccessor := texcoordTechCommon.CreateElement("accessor")
	texcoordAccessor.CreateAttr("source", "#ID8-array")
	texcoordAccessor.CreateAttr("count", fmt.Sprintf("%d", texcoordCount/2))
	texcoordAccessor.CreateAttr("stride", "2")

	sParam := texcoordAccessor.CreateElement("param")
	sParam.CreateAttr("name", "S")
	sParam.CreateAttr("type", "float")

	tParam := texcoordAccessor.CreateElement("param")
	tParam.CreateAttr("name", "T")
	tParam.CreateAttr("type", "float")

	// Vertices
	verticesElem := mesh.CreateElement("vertices")
	verticesElem.CreateAttr("id", "geometrySource1-vertices")

	positionInput := verticesElem.CreateElement("input")
	positionInput.CreateAttr("semantic", "POSITION")
	positionInput.CreateAttr("source", "#geometrySource1")

	normalsInput := verticesElem.CreateElement("input")
	normalsInput.CreateAttr("semantic", "NORMAL")
	normalsInput.CreateAttr("source", "#geometrySource2")

	// Triangles
	triangleCount := ((positionCount / 3) / 3)
	triangles := mesh.CreateElement("triangles")
	triangles.CreateAttr("count", fmt.Sprintf("%d", triangleCount))
	triangles.CreateAttr("material", "geometryElement5")

	vertexInput := triangles.CreateElement("input")
	vertexInput.CreateAttr("semantic", "VERTEX")
	vertexInput.CreateAttr("offset", "0")
	vertexInput.CreateAttr("source", "#geometrySource1-vertices")

	normalInput := triangles.CreateElement("input")
	normalInput.CreateAttr("semantic", "NORMAL")
	normalInput.CreateAttr("offset", "0")
	normalInput.CreateAttr("source", "#geometrySource2")

	texcoordInput := triangles.CreateElement("input")
	texcoordInput.CreateAttr("semantic", "TEXCOORD")
	texcoordInput.CreateAttr("offset", "0")
	texcoordInput.CreateAttr("source", "#geometrySource7")
	texcoordInput.CreateAttr("set", "1")

	triangles.CreateElement("p").CreateCharData(trianglesWriter.String())
}

func writeLibraryVisualScenes(parent *etree.Element, sceneID, nodeName, geometryID string) {
	libraryVisualScene := parent.CreateElement("library_visual_scenes")
	visualScene := libraryVisualScene.CreateElement("visual_scene")
	visualScene.CreateAttr("id", sceneID)

	nodeID := 1
	node := visualScene.CreateElement("node")
	node.CreateAttr("id", fmt.Sprintf("node%d", nodeID))
	node.CreateAttr("name", nodeName)

	instanceGeom := node.CreateElement("instance_geometry")
	instanceGeom.CreateAttr("url", fmt.Sprintf("#%s", geometryID))

	bindMaterial := instanceGeom.CreateElement("bind_material")
	bindMatTechCommon := bindMaterial.CreateElement("technique_common")
	instanceMat := bindMatTechCommon.CreateElement("instance_material")
	instanceMat.CreateAttr("symbol", "geometryElement5")
	instanceMat.CreateAttr("target", "#lambert1")

	bindVertexInput := instanceMat.CreateElement("bind_vertex_input")
	bindVertexInput.CreateAttr("semantic", "CHANNEL2")
	bindVertexInput.CreateAttr("input_semantic", "TEXCOORD")
	bindVertexInput.CreateAttr("input_set", "1")
}

func writeSceneElement(parent *etree.Element, name string) {
	scene := parent.CreateElement("scene")
	scene.CreateElement("instance_visual_scene").CreateAttr("url", fmt.Sprintf("#%s", name))
}
