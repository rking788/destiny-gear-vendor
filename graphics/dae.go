package graphics

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kpango/glg"

	"github.com/beevik/etree"
	"github.com/rking788/destiny-gear-vendor/bungie"
)

// A DAEWriter is responsible for writing the parsed object geometry to a Collada (.dae) file.
type DAEWriter struct {
	Path        string
	TexturePath string
}

// WriteModels will write the specified models into a single Collada (.dae) file.
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
	writeTextures(processed, dae.TexturePath)

	return nil
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
		zParam.CreateAttr("name", "Z")
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
		// 3 points per vertex, 3 vertices per triangle
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

	// It is important that all of the instance geometries are inside of the same node, that
	// makes it much easier to import into a scene later on without having to import each
	// ndoe/piece.
	nodeName := fmt.Sprintf("node-%s", geometryIDs[0])

	node := visualScene.CreateElement("node")
	node.CreateAttr("id", "node0")
	node.CreateAttr("name", nodeName)

	for i, geomID := range geometryIDs {

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
