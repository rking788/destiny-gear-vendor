package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/beevik/etree"
)

type DAEWriter struct {
	Path string
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
		texcoordsWriter.WriteString(fmt.Sprintf("%.9f ", coord))
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
