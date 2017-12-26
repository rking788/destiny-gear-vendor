package graphics

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/tidwall/gjson"
)

type DAEWriter struct {
	Path string
}

func (dae *DAEWriter) writeXML(positions []float64, texcoords []float32) error {

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
		texcoordsWriter.WriteString(fmt.Sprintf("%.6f ", coord))
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
	sampler2D.CreateElement("wrap_s").CreateCharData("WRAP")
	sampler2D.CreateElement("wrap_t").CreateCharData("WRAP")
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

	if includeTextures {
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
	}

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

func (dae *DAEWriter) WriteModels(geoms []*bungie.DestinyGeometry) error {

	positionVertices := make([]float64, 0, 10240)
	texcoords := make([]float32, 0, 10240)

	for _, geom := range geoms {
		result := gjson.Parse(string(geom.MeshesBytes))

		meshes := result.Get("render_model.render_meshes")
		if meshes.Exists() == false {
			err := errors.New("Error unmarshaling mesh JSON: render meshes not found")
			return err
		}

		fmt.Printf("Successfully parsed meshes JSON\n")

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

				data := geom.GetFileByName(vertexBuffers["file_name"].String()).Data
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
							v[l] = (positions[indexBuffer[start+j+tri[k]]][l] + offsetConstant) * scaleConstant
						}

						tex := [2]float32{}
						//fmt.Printf("Normal indices: ")
						for l := 0; l < 2; l++ {
							offset := texcoordOffset[l]
							scale := texcoordScale[l]
							adjustment := float32(1.0)
							if len(adjustments) > 0 {
								adjustment = adjustments[indexBuffer[start+j+tri[k]]][l]
							}
							//adjustment = 1.0
							//fmt.Printf("%d,", indexBuffer[start+j+tri[k]])
							tex[l] = (((texcoordVal(innerTexcoords[indexBuffer[start+j+tri[k]]][l])).normalize(16) * scale * adjustment) + offset) + manualOffsets[l]
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
	}

	dae.writeXML(positionVertices, texcoords)

	return nil
}
