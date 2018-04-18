package graphics

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"strings"
	"time"

	"github.com/kpango/glg"

	"github.com/beevik/etree"
	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/tidwall/gjson"
)

// A DAEWriter is responsible for writing the parsed object geometry to a Collada (.dae) file.
type DAEWriter struct {
	Path        string
	TexturePath string
}

func (dae *DAEWriter) writeTextures(processed *processedOutput) error {

	for _, plate := range processed.texturePlates {
		if plate == nil {
			continue
		}

		// Write this to a file now
		filePath := dae.TexturePath + "/" + plate.name
		glg.Infof("Writing texture file to: %s", filePath)
		outF, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			glg.Error(err)
			return err
		}

		flipped := flipVertically(plate.data)

		err = jpeg.Encode(outF, flipped, nil)
		if err != nil {
			glg.Error(err)
			return err
		}
		outF.Close()
	}

	return nil
}

func flipVertically(img image.Image) image.Image {
	dst := image.NewRGBA(img.Bounds())

	origin := img.Bounds().Min
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	for x := origin.X; x < (origin.X + width); x++ {
		for y := origin.Y; y < (origin.Y + height); y++ {
			newY := height - y
			dst.Set(x, newY, img.At(x, y))
		}
	}

	return dst
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

	writeLibraryImagesElement(colladaRoot, processed.texturePlates)

	writeLibraryEffects(colladaRoot, processed.texturePlates)

	writeLibraryMaterials(colladaRoot, processed.texturePlates)

	geometryIDs := writeLibraryGeometries(colladaRoot, processed)

	writeLibraryVisualScenes(colladaRoot, geometryIDs, processed)

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

func writeLibraryImagesElement(parent *etree.Element, plates [10]*texturePlate) {

	libImages := parent.CreateElement("library_images")

	for i, plate := range plates {
		if plate == nil {
			continue
		}
		img := libImages.CreateElement("image")
		libraryImagesID := fmt.Sprintf("image%d", i)
		img.CreateAttr("id", libraryImagesID)
		plate.libraryImagesID = libraryImagesID

		initFrom := img.CreateElement("init_from")
		initFrom.CreateCharData(fmt.Sprintf("%s", plate.name))
	}
}

func writeLibraryMaterials(parent *etree.Element, plates [10]*texturePlate) {

	libraryMaterials := parent.CreateElement("library_materials")

	for i, plate := range plates {
		if plate == nil {
			continue
		}

		materialID := fmt.Sprintf("lambert%d", i)
		plate.libraryMaterialID = materialID
		texMaterial := libraryMaterials.CreateElement("material")
		texMaterial.CreateAttr("id", materialID)
		texMaterial.CreateAttr("name", materialID)
		texMaterial.CreateElement("instance_effect").CreateAttr("url", fmt.Sprintf("#%s", plate.libraryEffectsID))
	}
}

func writeLibraryEffects(parent *etree.Element, plates [10]*texturePlate) {
	libraryEffects := parent.CreateElement("library_effects")

	// NOTE: I don't think this STL effect is used anymore, this was an effect for solid
	// colors before texturing was enabled. Mabyet this should be used when textures
	// are turned off?

	// effect := libraryEffects.CreateElement("effect")
	// effect.CreateAttr("id", materialEffectName)

	// profileCommon := effect.CreateElement("profile_COMMON")
	// technique := profileCommon.CreateElement("technique")
	// technique.CreateAttr("sid", "common")

	// phong := technique.CreateElement("phong")
	// ambient := phong.CreateElement("ambient")
	// ambient.CreateElement("color").CreateCharData("0 0 0 1")

	// diffuse := phong.CreateElement("diffuse")
	// diffuse.CreateElement("color").CreateCharData("1 1 1 1")

	// reflective := phong.CreateElement("reflective")
	// reflective.CreateElement("color").CreateCharData("0 0 0 1")

	// transparent := phong.CreateElement("transparent")
	// transparent.CreateAttr("opaque", "A_ONE")
	// transparent.CreateElement("color").CreateCharData("1 1 1 1")

	// transparency := phong.CreateElement("transparency")
	// transparency.CreateElement("float").CreateCharData("1")

	// indexOfRefraction := phong.CreateElement("index_of_refraction")
	// indexOfRefraction.CreateElement("float").CreateCharData("1")

	/**
	 * Lambert1 effects
	 **/

	for i, plate := range plates {
		if plate == nil {
			continue
		}

		lambertEffect := libraryEffects.CreateElement("effect")
		libraryEffectID := fmt.Sprintf("effect_lambert%d", i)
		lambertEffect.CreateAttr("id", libraryEffectID)
		plate.libraryEffectsID = libraryEffectID

		lambertProfileCommon := lambertEffect.CreateElement("profile_COMMON")

		imageSurfaceSourceID := fmt.Sprintf("ID2_image%d_surface", i)
		imgSurfNewParam := lambertProfileCommon.CreateElement("newparam")
		imgSurfNewParam.CreateAttr("sid", imageSurfaceSourceID)

		surface := imgSurfNewParam.CreateElement("surface")
		surface.CreateAttr("type", "2D")
		surface.CreateElement("init_from").CreateCharData(plate.libraryImagesID)

		imageSID := fmt.Sprintf("ID2_image%d", i)
		imageNewParam := lambertProfileCommon.CreateElement("newparam")
		imageNewParam.CreateAttr("sid", imageSID)

		sampler2D := imageNewParam.CreateElement("sampler2D")
		sampler2D.CreateElement("source").CreateCharData(imageSurfaceSourceID)
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
			diffuseTexture.CreateAttr("texture", imageSID)
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
		triangles.CreateAttr("material", geometryID)

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

func writeLibraryVisualScenes(parent *etree.Element, geometryIDs []string, processed *processedOutput) {

	sceneID := 1
	sceneName := fmt.Sprintf("scene%d", sceneID)

	libraryVisualScene := parent.CreateElement("library_visual_scenes")
	visualScene := libraryVisualScene.CreateElement("visual_scene")
	visualScene.CreateAttr("id", sceneName)

	glg.Warnf("Starting with %d geometry IDs", len(geometryIDs))
	glg.Warnf("Starting with %+v plateIndices", processed.plateIndices)
	for i, geomID := range geometryIDs {

		nodeName := fmt.Sprintf("node-%s", geomID)
		nodeID := i

		node := visualScene.CreateElement("node")
		node.CreateAttr("id", fmt.Sprintf("node%d", nodeID))
		node.CreateAttr("name", nodeName)

		instanceGeom := node.CreateElement("instance_geometry")
		instanceGeom.CreateAttr("url", fmt.Sprintf("#%s", geomID))

		glg.Infof("GeomIndex: %d", i)
		glg.Infof("GoemID: %s", geomID)
		if i < len(processed.plateIndices) {
			plateIndex := processed.plateIndices[i]
			glg.Infof("Found plate index = %d", plateIndex)
			if plateIndex != -1 && plateIndex < len(processed.texturePlates) {
				bindMaterial := instanceGeom.CreateElement("bind_material")
				bindMatTechCommon := bindMaterial.CreateElement("technique_common")
				instanceMat := bindMatTechCommon.CreateElement("instance_material")

				texturePlate := processed.texturePlates[plateIndex]
				glg.Infof("Found texture plate material = %s", texturePlate.libraryMaterialID)

				instanceMat.CreateAttr("symbol", geomID)
				instanceMat.CreateAttr("target", fmt.Sprintf("#%s", texturePlate.libraryMaterialID))

				bindVertexInput := instanceMat.CreateElement("bind_vertex_input")
				bindVertexInput.CreateAttr("semantic", "CHANNEL2")
				bindVertexInput.CreateAttr("input_semantic", "TEXCOORD")
				bindVertexInput.CreateAttr("input_set", "1")
			}
		}
	}

	scene := parent.CreateElement("scene")
	scene.CreateElement("instance_visual_scene").CreateAttr("url", fmt.Sprintf("#%s", sceneName))
}

type texturePlate struct {
	name string
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
	plateIndices  []int
	texturePlates [10]*texturePlate
}

func (dae *DAEWriter) WriteModels(geoms []*bungie.DestinyGeometry) error {

	geomCount := len(geoms)
	processed := &processedOutput{
		positionVertices: make([][]float64, 0, geomCount),
		normalValues:     make([][]float64, 0, geomCount),
		texcoords:        make([][]float32, 0, geomCount),
		plateIndices:     make([]int, 0, geomCount),
	}

	glg.Infof("Starting with output struct: %+v", processed)

	glg.Infof("Writing models for %d geometries", len(geoms))

	for _, geom := range geoms {
		err := processGeometry(geom, processed)
		if err != nil {
			glg.Errorf("Failed to process Bungie geometry object: %s", err.Error())
			return err
		}
	}

	glg.Warnf("Positions count = %d;;;Plate indices count = %d", len(processed.positionVertices), len(processed.plateIndices))
	dae.writeXML(processed)
	dae.writeTextures(processed)

	return nil
}

func processGeometry(geom *bungie.DestinyGeometry, output *processedOutput) error {
	result := gjson.Parse(string(geom.MeshesBytes))

	startPosCount := len(output.positionVertices)
	// Process render meshes
	meshes := result.Get("render_model.render_meshes")
	if meshes.Exists() == false {
		return errors.New("Error unmarshaling mesh JSON: render meshes not found")
	}

	glg.Info("Successfully parsed meshes JSON")

	meshArray := meshes.Array()
	glg.Infof("Found %d meshes", len(meshArray))

	for _, meshInterface := range meshArray {

		mesh := meshInterface.Map()

		err := processMesh(mesh, output, geom.GetFileByName)
		if err != nil {
			return err
		}
	}

	// Process textures
	plates := result.Get("texture_plates")
	if plates.Exists() == false {
		return errors.New("Error unmarshaling render model JSON: texture plates not found")
	}

	glg.Info("Successfully parsed plates JSON")

	platesArray := plates.Array()
	glg.Infof("Found %d plates", len(platesArray))
	if len(platesArray) > 1 {
		panic("Found more than 1 texture plate in this render.json")
	} else if len(platesArray) <= 0 {
		glg.Warnf("Found 0 plates in this render file")
		return nil
	}

	newGeomCount := len(output.positionVertices) - startPosCount
	processTexturePlate(platesArray[0].Map(), newGeomCount, output)

	glg.Infof("Position length : %d", len(output.positionVertices))
	glg.Infof("Found plate indices : %+v", output.plateIndices)

	return nil
}

func processTexturePlate(texturePlateJSON map[string]gjson.Result, newGeomElements int, output *processedOutput) error {

	diffuseSet := texturePlateJSON["plate_set"].Get("diffuse")
	plateIndex := int(diffuseSet.Get("plate_index").Int())
	plateSizeTemp := diffuseSet.Get("plate_size").Array()
	plateSize := [2]int{int(plateSizeTemp[0].Int()), int(plateSizeTemp[1].Int())}

	texturePlacements := diffuseSet.Get("texture_placements").Array()

	// if len(texturePlacements) <= 0 {
	// 	glg.Info("Found 0 texutre placements for this geometry... skipping.")
	// 	//output.plateIndices = append(output.plateIndices, -1)
	// 	return nil
	// }

	for i := 0; i < newGeomElements; i++ {
		// Use this texture plate for all newly added geometries
		output.plateIndices = append(output.plateIndices, plateIndex)
	}

	sizeX := plateSize[0]
	sizeY := plateSize[1]
	posX := 0
	posY := 0
	textureTagName := fmt.Sprintf("blank-%d", plateIndex)
	if len(texturePlacements) > 0 {
		placement := texturePlacements[0]
		sizeX = int(placement.Get("texture_size_x").Int())
		sizeY = int(placement.Get("texture_size_y").Int())
		posX = int(placement.Get("position_x").Int())
		posY = int(placement.Get("position_y").Int())
		textureTagName = placement.Get("texture_tag_name").String()
	}

	plate := output.texturePlates[plateIndex]
	if plate == nil {

		plate = &texturePlate{
			name: textureTagName + ".jpg",
			size: plateSize,
			data: blackImageOfSize(plateSize),
		}
		output.texturePlates[plateIndex] = plate
	}

	if len(texturePlacements) <= 0 {
		// No textures to plate, just leave it as a black image
		return nil
	}

	// Copy this data into the destination image
	textureImgPath := "./output/" + textureTagName + ".jpg"
	inF, err := os.OpenFile(textureImgPath, os.O_RDONLY, 0644)
	if err != nil {
		glg.Error(err)
		return err
	}

	img, format, err := image.Decode(inF)
	if err != nil {
		glg.Error(err)
		return err
	}

	glg.Debugf("Successfully decoded image with format: %s", format)

	dp := image.Point{posX, posY}
	r := image.Rectangle{dp, dp.Add(image.Point{sizeX, sizeY})}
	draw.Draw(plate.data, r, img, image.ZP, draw.Src)

	return nil
}

func blackImageOfSize(size [2]int) draw.Image {
	img := image.NewRGBA(image.Rect(0, 0, size[0], size[1]))
	black := color.RGBA{0, 0, 0, 255}

	draw.Draw(img, img.Bounds(), image.NewUniform(black), image.ZP, draw.Src)

	return img
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
					glg.Debugf("Found adjustments: len=%d", len(adjustmentsVb))
				}
			default:
				glg.Warnf("Unhandled semantic: %s", element["semantic"].String())
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

		texcoordOffsetsTemp := mesh["texcoord_offset"].Array()
		texcoordScalesTemp := mesh["texcoord_scale"].Array()

		texcoordOffsets := [2]float64{texcoordOffsetsTemp[0].Float(), texcoordOffsetsTemp[1].Float()}
		texcoordScales := [2]float64{texcoordScalesTemp[0].Float(), texcoordScalesTemp[1].Float()}

		glg.Debugf("Found texcoord offsets: %+v", texcoordOffsets)
		glg.Debugf("Found texcoord scales: %+v", texcoordScales)

		processPart(part, i, indexBuffer, positionsVb, normalsVb, innerTexcoordsVb, adjustmentsVb, texcoordOffsets, texcoordScales, output)
	}

	return nil
}

func processPart(part map[string]gjson.Result, partIndex int, indexBuffer []int16, positionsVb, normalsVb [][]float64, innerTexcoordsVb, adjustmentsVb [][]float32, texcoordOffsets, texcoordScales [2]float64, output *processedOutput) error {

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

				texcoordIndex := indexBuffer[start+j+tri[k]]

				tex[l] = transformTexcoord(innerTexcoordsVb[texcoordIndex], l, texcoordOffsets[l], texcoordScales[l])
			}

			// Positions, Normals, Texture coordinates for the processed "part"
			pos = append(pos, v[0], v[1], v[2])
			norm = append(norm, n[0], n[1], n[2])
			texcoords = append(texcoords, tex[0], tex[1])
		}
	}

	glg.Warnf("Appending position vertices")
	output.positionVertices = append(output.positionVertices, pos)
	output.normalValues = append(output.normalValues, norm)
	output.texcoords = append(output.texcoords, texcoords)

	return nil
}

func transformTexcoord(coords []float32, index int, offset, scale float64) float32 {

	//glg.Debugf("Using coord(%f) offset(%f) and scale(%f)",
	//	texcoordVal(coords[index]).normalize(16), offset, scale)
	return ((texcoordVal(coords[index]).normalize(16) * float32(scale)) + float32(offset))
}
