package graphics

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kpango/glg"

	"github.com/beevik/etree"
	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/tidwall/gjson"
)

const (
	textureFilename = "crimson-diffuse.jpg"
)

// A DAEWriter is responsible for writing the parsed object geometry to a Collada (.dae) file.
type DAEWriter struct {
	Path string
}

func (dae *DAEWriter) writeXML(processed *processedOutput) error {

	if len(processed.positionVertices) <= 0 {
		return errors.New("Empty position vertices, nothing to do here")
	} else if len(processed.positionVertices) != len(processed.normalValues) ||
		len(processed.positionVertices) != len(processed.texcoords) {
		return errors.New("Mismatched number of position, normals, or texcoords")
	}

	doc, colladaRoot := NewColladaDoc()

	writeAssetElement(colladaRoot)
	writeImageLibraryElement(colladaRoot)

	// TODO: These cannot be empty, need to add solid material data
	materialID := "STL_material"
	materialEffectName := "effect_STL_material"
	writeLibraryMaterials(colladaRoot, materialID, materialEffectName)

	writeLibraryEffects(colladaRoot, materialEffectName)

	geometryIDs := writeLibraryGeometries(colladaRoot, processed)

	writeLibraryVisualScenes(colladaRoot, geometryIDs)

	doc.Indent(2)
	//doc.WriteTo(os.Stdout)

	// Write this to a file now
	outF, err := os.OpenFile(dae.Path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		glg.Error(err)
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

	imgName := textureFilename
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

	if includeTextures {
		diffuseTexture := blinnDiffuse.CreateElement("texture")
		diffuseTexture.CreateAttr("texture", "ID2_image1")
		diffuseTexture.CreateAttr("texcoord", "CHANNEL2")
	} else {
		blinnDiffuse.CreateElement("color").CreateCharData("1 0.7 0.5 1")
	}
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

func writeLibraryGeometries(parent *etree.Element, processed *processedOutput) []string {

	libGeometries := parent.CreateElement("library_geometries")

	geometryIDs := make([]string, 0, len(processed.positionVertices))

	includedIndices := []int{}
	for i := range processed.positionVertices {

		// NOTE: This is only for debugging and in the case where it's helpful to
		// break up the whole item into individual geometries.
		if len(includedIndices) != 0 {

			found := false
			for _, index := range includedIndices {
				if index == i {
					found = true
				}
			}

			if !found {
				glg.Debug("continuing")
				continue
			}
		}

		geometryID := fmt.Sprintf("3054293897-0_%d_1", i*3+1)
		posSourceID := fmt.Sprintf("geometrySource%d", i*3+1)
		posFloatArrayID := fmt.Sprintf("ID%d-array", i*3+2)
		normalsSourceID := fmt.Sprintf("geometrySource%d", i*3+2)
		normalsFloatArrayID := fmt.Sprintf("ID%d-array", i*3+2)
		texcoordSourceID := fmt.Sprintf("geometrySource%d", i*3+3)
		texcoordFloatArrayID := fmt.Sprintf("ID%d-array", i*3+3)
		posVerticesID := fmt.Sprintf("%s-vertices", posSourceID)

		currentPositions := processed.positionVertices[i]
		currentNormals := processed.normalValues[i]
		currentTexcoords := processed.texcoords[i]

		positionCount := len(currentPositions)
		//normalCount := len(currentNormals)
		texcoordCount := len(currentTexcoords)

		posWriter := bytes.NewBufferString("")
		normalWriter := bytes.NewBufferString("")
		trianglesWriter := bytes.NewBufferString("")
		texcoordWriter := bytes.NewBufferString("")

		// TODO: Step3, this should not flatten out the 2D slices, it should put each slice
		// into their own string
		glg.Debugf("Pos list length = %d", len(currentPositions))
		for i, pos := range currentPositions {
			posWriter.WriteString(fmt.Sprintf("%f ", pos))
			trianglesWriter.WriteString(fmt.Sprintf("%d ", i))
		}

		glg.Debugf("Wrote %d positions to the DAE file", len(currentPositions))

		for _, norm := range currentNormals {
			normalWriter.WriteString(fmt.Sprintf("%f ", norm))
		}

		for _, coord := range currentTexcoords {
			texcoordWriter.WriteString(fmt.Sprintf("%.6f ", coord))
		}

		// TODO: This should actually be read from the DestinyGeometry type
		geometry := libGeometries.CreateElement("geometry")
		geometry.CreateAttr("id", geometryID)
		geometry.CreateAttr("name", geometryID)

		mesh := geometry.CreateElement("mesh")

		posSource := mesh.CreateElement("source")
		posSource.CreateAttr("id", posSourceID)

		posFloatArray := posSource.CreateElement("float_array")
		posFloatArray.CreateAttr("id", posFloatArrayID)
		posFloatArray.CreateAttr("count", fmt.Sprintf("%d", positionCount))
		posFloatArray.CreateCharData(strings.TrimSpace(posWriter.String()))

		techniqueCommon := posSource.CreateElement("technique_common")
		accessor := techniqueCommon.CreateElement("accessor")
		accessor.CreateAttr("source", fmt.Sprintf("#%s", posFloatArrayID))
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
		normalsSource.CreateAttr("id", normalsSourceID)

		normFloatArray := normalsSource.CreateElement("float_array")
		normFloatArray.CreateAttr("id", normalsFloatArrayID)
		normFloatArray.CreateAttr("count", fmt.Sprintf("%d", positionCount))
		normFloatArray.CreateCharData(strings.TrimSpace(normalWriter.String()))

		normTechniqueCommon := normalsSource.CreateElement("technique_common")
		normAccessor := normTechniqueCommon.CreateElement("accessor")
		normAccessor.CreateAttr("source", fmt.Sprintf("#%s", normalsFloatArrayID))
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
			texcoordSource.CreateAttr("id", texcoordSourceID)

			texcoordFloatArray := texcoordSource.CreateElement("float_array")
			texcoordFloatArray.CreateAttr("id", texcoordFloatArrayID)
			texcoordFloatArray.CreateAttr("count", fmt.Sprintf("%d", texcoordCount))
			texcoordFloatArray.CreateCharData(strings.TrimSpace(texcoordWriter.String()))

			texcoordTechCommon := texcoordSource.CreateElement("technique_common")
			texcoordAccessor := texcoordTechCommon.CreateElement("accessor")
			texcoordAccessor.CreateAttr("source", fmt.Sprintf("#%s", texcoordFloatArrayID))
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
		verticesElem.CreateAttr("id", posVerticesID)

		positionInput := verticesElem.CreateElement("input")
		positionInput.CreateAttr("semantic", "POSITION")
		positionInput.CreateAttr("source", fmt.Sprintf("#%s", posSourceID))

		normalsInput := verticesElem.CreateElement("input")
		normalsInput.CreateAttr("semantic", "NORMAL")
		normalsInput.CreateAttr("source", fmt.Sprintf("#%s", normalsSourceID))

		// Triangles
		triangleCount := ((positionCount / 3) / 3)
		triangles := mesh.CreateElement("triangles")
		triangles.CreateAttr("count", fmt.Sprintf("%d", triangleCount))
		triangles.CreateAttr("material", "geometryElement5")

		vertexInput := triangles.CreateElement("input")
		vertexInput.CreateAttr("semantic", "VERTEX")
		vertexInput.CreateAttr("offset", "0")
		vertexInput.CreateAttr("source", fmt.Sprintf("#%s", posVerticesID))

		normalInput := triangles.CreateElement("input")
		normalInput.CreateAttr("semantic", "NORMAL")
		normalInput.CreateAttr("offset", "0")
		normalInput.CreateAttr("source", fmt.Sprintf("#%s", normalsSourceID))

		texcoordInput := triangles.CreateElement("input")
		texcoordInput.CreateAttr("semantic", "TEXCOORD")
		texcoordInput.CreateAttr("offset", "0")
		texcoordInput.CreateAttr("source", fmt.Sprintf("#%s", texcoordSourceID))
		texcoordInput.CreateAttr("set", "1")

		triangles.CreateElement("p").CreateCharData(trianglesWriter.String())

		geometryIDs = append(geometryIDs, geometryID)
	}

	return geometryIDs
}

func writeLibraryVisualScenes(parent *etree.Element, geometryIDs []string) {

	sceneID := 1
	sceneName := fmt.Sprintf("scene%d", sceneID)

	libraryVisualScene := parent.CreateElement("library_visual_scenes")
	visualScene := libraryVisualScene.CreateElement("visual_scene")
	visualScene.CreateAttr("id", sceneName)

	for i, geomID := range geometryIDs {

		nodeName := fmt.Sprintf("node-%s", geomID)
		nodeID := i

		node := visualScene.CreateElement("node")
		node.CreateAttr("id", fmt.Sprintf("node%d", nodeID))
		node.CreateAttr("name", nodeName)

		instanceGeom := node.CreateElement("instance_geometry")
		instanceGeom.CreateAttr("url", fmt.Sprintf("#%s", geomID))

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

	scene := parent.CreateElement("scene")
	scene.CreateElement("instance_visual_scene").CreateAttr("url", fmt.Sprintf("#%s", sceneName))
}

type processedOutput struct {
	// One slice of floats for each mesh (parts are all included in a single slice)
	positionVertices [][]float64
	normalValues     [][]float64
	texcoords        [][]float32
}

func (dae *DAEWriter) WriteModels(geoms []*bungie.DestinyGeometry) error {

	processed := &processedOutput{
		positionVertices: make([][]float64, 0, 10),
		normalValues:     make([][]float64, 0, 10),
		texcoords:        make([][]float32, 0, 10),
	}

	glg.Infof("Writing models for %d geometries", len(geoms))

	for _, geom := range geoms {
		err := processGeometry(geom, processed)
		if err != nil {
			glg.Errorf("Failed to process Bungie geometry object: %s", err.Error())
			return err
		}
	}

	dae.writeXML(processed)

	return nil
}

func processGeometry(geom *bungie.DestinyGeometry, output *processedOutput) error {
	result := gjson.Parse(string(geom.MeshesBytes))

	meshes := result.Get("render_model.render_meshes")
	if meshes.Exists() == false {
		err := errors.New("Error unmarshaling mesh JSON: render meshes not found")
		return err
	}

	glg.Info("Successfully parsed meshes JSON")

	meshArray := meshes.Array()
	glg.Infof("Found %d meshes", len(meshArray))

	for meshIndex, meshInterface := range meshArray {
		if meshIndex != 1 {
			//continue
		}

		mesh := meshInterface.Map()

		err := processMesh(mesh, output, geom.GetFileByName)
		if err != nil {
			return err
		}
	}

	return nil
}

func processMesh(mesh map[string]gjson.Result, output *processedOutput, fileProvider func(string) *bungie.GeometryFile) error {

	positionsVb := [][]float64{}
	normalsVb := [][]float64{}
	innerTexcoordsVb := [][]float32{}
	adjustmentsVb := [][]float32{}

	defVB := mesh["stage_part_vertex_stream_layout_definitions"].Array()[0].Map()["formats"].Array()

	for index, vbInterface := range mesh["vertex_buffers"].Array() {

		currentDefVB := defVB[index].Map()
		vertexBuffers := vbInterface.Map()
		stride := currentDefVB["stride"].Float()
		if stride != vertexBuffers["stride_byte_size"].Float() {
			return errors.New("Mismatched stride sizes found")
		}

		data := fileProvider(vertexBuffers["file_name"].String()).Data
		glg.Infof("Reading data from file: %s", vertexBuffers["file_name"].String())
		if data == nil {
			return errors.New("Missing geometry file by name: " + vertexBuffers["file_name"].String())
		}

		for _, elementInterface := range currentDefVB["elements"].Array() {
			element := elementInterface.Map()
			elementType := element["type"].String()
			elementOffset := element["offset"].Float()

			switch element["semantic"].String() {
			case "_tfx_vb_semantic_position":
				positionsVb = parseVertex(data, elementType, int(elementOffset), int(stride))
				glg.Debugf("Found positions: %d", len(positionsVb))
			case "_tfx_vb_semantic_normal":
				normalsVb = parseVertex(data, elementType, int(elementOffset), int(stride))
				glg.Debugf("Found normals: len=%d", len(normalsVb))
			case "_tfx_vb_semantic_texcoord":
				if elementType != "_vertex_format_attribute_float2" {
					innerTexcoordsVb = parseVertex32(data, elementType, int(elementOffset), int(stride))
					glg.Debugf("Found textcoords: len=%d", len(innerTexcoordsVb))
				} else {
					adjustmentsVb = parseVertex32(data, elementType, int(elementOffset), int(stride))
					glg.Debugf("Found adjustments: %d", len(adjustmentsVb))
					//fmt.Printf("Found float texcoords: %+v\n", throwaway)
				}
			}
		}
	}

	if len(positionsVb) == 0 || len(normalsVb) == 0 || len(positionsVb) != len(normalsVb) {
		return errors.New("Positions slice is not the same size as the normals slice")
	}

	// Parse the index buffer
	indexBuffer := make([]int16, 0)
	indexBufferBytes := fileProvider(mesh["index_buffer"].Get("file_name").String()).Data

	for i := 0; i < len(indexBufferBytes); i += 2 {

		var index int16
		binary.Read(bytes.NewBuffer(indexBufferBytes[i:i+2]), binary.LittleEndian, &index)
		indexBuffer = append(indexBuffer, index)
	}

	parts := mesh["stage_part_list"].Array()
	glg.Infof("Found %d stage parts", len(parts))

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

		processPart(part, i, indexBuffer, positionsVb, normalsVb, innerTexcoordsVb, adjustmentsVb, output)
	}

	return nil
}

func processPart(part map[string]gjson.Result, partIndex int, indexBuffer []int16, positionsVb, normalsVb [][]float64, innerTexcoordsVb, adjustmentsVb [][]float32, output *processedOutput) error {

	start := int(part["start_index"].Float())
	count := int(part["index_count"].Float())

	pos := make([]float64, 0, 1024)
	norm := make([]float64, 0, 1024)
	texcoords := make([]float32, 0, 1024)

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
		glg.Warn("Unknown primitive type, skipping this part...")
		// Don't throw an error, just return nil so this part is skipped. continue
		// on to the next part
		return nil
	}

	// Construct and write this mesh header
	for j := 0; j < count; j += increment {

		// Skip if any two of the indexBuffer match (ignoring lines or points)
		if indexBuffer[start+j+0] == indexBuffer[start+j+1] ||
			indexBuffer[start+j+0] == indexBuffer[start+j+2] ||
			indexBuffer[start+j+1] == indexBuffer[start+j+2] {
			continue
		}

		tri := [3]int{0, 1, 2}
		if (primitiveType == 3) || ((j & 1) == 1) {
			tri = [3]int{2, 1, 0}
		}

		for k := 0; k < 3; k++ {
			v := [4]float64{}
			n := [4]float64{}
			for l := 0; l < 4; l++ {
				index := start + j + tri[k]
				if index >= len(indexBuffer) {
					// TODO: These should be converted to uint16 so the indices don't seem to be out of bounds
					glg.Errorf("*** ERROR: Current Index is outside the bounds of the index buffer: Want=%d, Actual=%d", index, len(indexBuffer))
					return errors.New("Current index outside bounds of indx buffer")
					//continue
				} else if uint(indexBuffer[index]) >= uint(len(positionsVb)) {
					// TODO: These should be converted to uint16 so the indices don't seem to be out of bounds
					glg.Errorf("*** ERROR: Current index buffer value is outside the bounds of the positions array: Want=%d, Actual=%d", indexBuffer[index], len(positionsVb))
					return errors.New("Current index buffer value is outside the bounds of the positions array")
					//continue
				}

				positionIndex := uint(indexBuffer[index])
				if l >= len(positionsVb[positionIndex]) {

					// TODO: These should be converted to uint16 so the indices don't seem to be out of bounds
					glg.Errorf("*** ERROR: Triangle index is outside the bounds of the current position array.")
					return errors.New("Current Triangle index outside teh bounds of the current position array")
					//continue
				}

				v[l] = positionsVb[positionIndex][l]
				n[l] = normalsVb[positionIndex][l]
			}

			tex := [2]float32{}

			for l := 0; l < 2; l++ {

				adjustment := float32(1.0)
				texcoordIndex := indexBuffer[start+j+tri[k]]
				if len(adjustmentsVb) > 0 {
					adjustment = adjustmentsVb[texcoordIndex][l]
				}

				tex[l] = transformTexcoord(innerTexcoordsVb[texcoordIndex], l, adjustment)
			}

			// Positions, Normals, Texture coordinates for the processed "part"
			pos = append(pos, v[0], v[1], v[2])
			norm = append(norm, n[0], n[1], n[2])
			texcoords = append(texcoords, tex[0], tex[1])
		}
	}

	output.positionVertices = append(output.positionVertices, pos)
	output.normalValues = append(output.normalValues, norm)
	output.texcoords = append(output.texcoords, texcoords)

	return nil
}

func transformTexcoord(coords []float32, index int, adjustment float32) float32 {
	offset := texcoordOffset[index]
	scale := texcoordScale[index]

	return (((texcoordVal(coords[index])).normalize(16) * scale * adjustment) + offset) + manualOffsets[index]
	//return ((float32(texcoordVal(coords[index])) * scale) + offset)
}
