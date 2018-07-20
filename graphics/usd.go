package graphics

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kpango/glg"
	"github.com/rking788/destiny-gear-vendor/bungie"
)

const (
	// PositionScaleConstant is a constant value that position vertices should be multipled
	// by to get a better size in augmented reality.
	PositionScaleConstant = 100.0

	// includePBRTextures is a flag to turn off the ambient occlusion, metalness, roughness
	// being written to the output USD file.
	includePBRTextures = true
)

// USDWriter is responsible for writing the parsed object geometry to a
// Univsersal Scene Description (.usd) file.
type USDWriter struct {
	Path        string
	TexturePath string
	output      io.Writer
}

// WriteModel will take the provided Destiny geometries and write them to a new file
// in the USD format.
func (usd *USDWriter) WriteModel(geoms []*bungie.DestinyGeometry) error {

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
	usd.write(processed)
	writeTextures(processed, usd.TexturePath)

	return nil
}

func (usd *USDWriter) write(processed *processedOutput) error {

	if len(processed.positionVertices) <= 0 {
		return errors.New("Empty position vertices, nothing to do here")
	} else if len(processed.positionVertices) != len(processed.normalValues) ||
		len(processed.positionVertices) != len(processed.texcoords) {
		return errors.New("Mismatched number of position, normals, or texcoords")
	}

	var err error
	usd.output, err = NewUSDDoc(usd.Path)
	if err != nil {
		return err
	}

	usd.writeMaterials(processed)
	usd.writeXforms(processed)

	return nil
}

// NewUSDDoc is a helper method for opening an io.Writer that can be used
// to write the contents of the USD file. This will also write the appropriate header metadata.
func NewUSDDoc(path string) (io.Writer, error) {

	outF, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	_, err = outF.Write([]byte(`#usda 1.0
(
    doc = """Generated from the Destiny Gear Vendor"""

    endTimeCode = 200
    startTimeCode = 1
    timeCodesPerSecond = 24
    upAxis = "Z"
)

`))

	return outF, err
}

func (usd *USDWriter) writeMaterials(processed *processedOutput) error {

	_, err := usd.output.Write([]byte("def Scope \"Materials\"\n{\n"))

	// for mat := materials {
	for i, plate := range processed.texturePlates {
		if plate == nil {
			continue
		}

		glg.Infof("Reading pbr index: %d", i)
		glg.Infof("Writing texture plate with name: %s", plate.name)
		// TODO: These can be consolidated into a method on the gearstack texture plate type
		gearstackName := processed.gearstackTexturePlates[i].name
		aoName := strings.Replace(gearstackName, "gearstack", "AO", -1)
		metalnessName := strings.Replace(gearstackName, "gearstack", "metalness", -1)
		roughnessName := strings.Replace(gearstackName, "gearstack", "roughness", -1)
		emissiveName := strings.Replace(gearstackName, "gearstack", "emissive", -1)

		plate.libraryMaterialID = fmt.Sprintf("Material%d", i)
		err = usd.writeMaterial(plate.libraryMaterialID, plate.name, aoName, metalnessName, roughnessName, emissiveName)
	}

	_, err = usd.output.Write([]byte(`    def Material "lambert1"
    {
        color3f inputs:displayColor = (0.5, 0.5, 0.5)
    }
}

`))

	return err
}

func (usd *USDWriter) writeMaterial(matID, albedoFilename, aoName, metalnessName, roughnessName, emissiveName string) error {

	_, err := usd.output.Write([]byte(`    def Material "` + matID + `"
	{
		token inputs:frame:stPrimvarName = "Texture_uv"
		token outputs:displacement.connect = </Materials/` + matID + `/pbrMat.outputs:displacement>
		token outputs:surface.connect = </Materials/` + matID + `/pbrMat.outputs:surface>

		def Shader "pbrMat"
		{
			uniform token info:id = "UsdPreviewSurface"
			float inputs:clearcoat = 0
			float inputs:clearcoatRoughness = 0
			color3f inputs:diffuseColor.connect = </Materials/` + matID + `/color_map.outputs:rgb>
			float inputs:displacement = 0
			color3f inputs:emissive.connect = </Materials/` + matID + `/emissive_map.outputs:r>
			float inputs:ior = 1.5
			float inputs:metallic.connect = </Materials/` + matID + `/metallic_map.outputs:r>
			normal3f inputs:normal.connect = </Materials/` + matID + `/normal_map.outputs:rgb>
			float inputs:occlusion.connect = </Materials/` + matID + `/ao_map.outputs:r>
			float inputs:opacity = 1
			float inputs:roughness.connect = </Materials/` + matID + `/roughness_map.outputs:r>
			color3f inputs:specularColor = (1, 1, 1)
			int inputs:useSpecularWorkflow = 0
			token outputs:displacement
			token outputs:surface
		}

		def Shader "color_map"
		{
			uniform token info:id = "UsdUVTexture"
			float4 inputs:default = (0, 0, 0, 1)
			asset inputs:file = @` + albedoFilename + `@
			float2 inputs:st.connect = </Materials/` + matID + `/Primvar.outputs:result>
			token inputs:wrapS = "repeat"
			token inputs:wrapT = "repeat"
			float3 outputs:rgb
		}

		def Shader "Primvar"
		{
			uniform token info:id = "UsdPrimvarReader_float2"
			float2 inputs:default = (0, 0)
			token inputs:varname.connect = </Materials/` + matID + `.inputs:frame:stPrimvarName>
			float2 outputs:result
		}

`))

	if !includePBRTextures {
		usd.output.Write([]byte(`    }
`))
	} else {
		usd.output.Write([]byte(`
        def Shader "ao_map"
        {
            uniform token info:id = "UsdUVTexture"
            float4 inputs:default = (0, 0, 0, 1)
            asset inputs:file = @` + aoName + `@
            float2 inputs:st.connect = </Materials/` + matID + `/Primvar.outputs:result>
            token inputs:wrapS = "repeat"
            token inputs:wrapT = "repeat"
            float outputs:r
        }

		def Shader "metallic_map"
        {
            uniform token info:id = "UsdUVTexture"
            float4 inputs:default = (0, 0, 0, 1)
            asset inputs:file = @` + metalnessName + `@
            float2 inputs:st.connect = </Materials/` + matID + `/Primvar.outputs:result>
            token inputs:wrapS = "repeat"
            token inputs:wrapT = "repeat"
            float outputs:r
        }

        def Shader "roughness_map"
        {
            uniform token info:id = "UsdUVTexture"
            float4 inputs:default = (0, 0, 0, 1)
            asset inputs:file = @` + roughnessName + `@
            float2 inputs:st.connect = </Materials/` + matID + `/Primvar.outputs:result>
            token inputs:wrapS = "repeat"
            token inputs:wrapT = "repeat"
            float outputs:r
		}
		
		def Shader "emissive_map"
		{
			uniform token info:id = "UsdUVTexture"
            float4 inputs:default = (0, 0, 0, 1)
            asset inputs:file = @` + emissiveName + `@
            float2 inputs:st.connect = </Materials/` + matID + `/Primvar.outputs:result>
            token inputs:wrapS = "repeat"
            token inputs:wrapT = "repeat"
            float outputs:r
		}
	}

`))
	}
	return err
}

func (usd *USDWriter) writeXforms(processed *processedOutput) error {

	usd.output.Write([]byte("def Xform \"Crimson\"\n{"))

	for i := range processed.positionVertices {
		usd.writeMesh(i, processed)
	}

	usd.output.Write([]byte("}"))

	return nil
}

func (usd *USDWriter) writeMesh(meshIndex int, processed *processedOutput) error {

	currentPositions := processed.positionVertices[meshIndex]
	currentNormals := processed.normalValues[meshIndex]
	currentTexcoords := processed.texcoords[meshIndex]

	positionCount := len(currentPositions)
	normalCount := len(currentNormals)
	texcoordCount := len(currentTexcoords)
	triangleCount := ((positionCount / 3) / 3)

	materialID := "lambert1"

	if meshIndex < len(processed.plateIndices) {
		glg.Debugf("Plate Indices: %+v", processed.plateIndices)
		plateIndex := processed.plateIndices[meshIndex]
		glg.Debugf("TexturePlates: %+v", processed.texturePlates)
		plate := processed.texturePlates[plateIndex]
		if plate != nil {
			materialID = processed.texturePlates[plateIndex].libraryMaterialID
		}
	}

	glg.Infof("Triangle Count: %d", triangleCount)
	faceVertexCounts := make([]string, 0, triangleCount)
	for i := 0; i < triangleCount; i++ {
		faceVertexCounts = append(faceVertexCounts, "3")
	}

	/**
	 * OPENING ITEM GEOM MESH + MATERIAL
	 */
	usd.output.Write([]byte(`
    def Mesh "CrimsonPiece` + fmt.Sprintf("%d\"", meshIndex) + `
    {
`))

	/**
	 * FACE VERTEX COUNTS *
	 */
	faceVertexCountArray := strings.Join(faceVertexCounts, ", ")
	usd.output.Write([]byte(fmt.Sprintf("        int[] faceVertexCounts = [%s]\n",
		faceVertexCountArray)))

	/**
	 * FACE VERTEX INDICES
	 */
	vertexIndices := make([]string, 0, len(currentPositions))
	for i := 0; i < len(currentPositions)/3; i++ {
		vertexIndices = append(vertexIndices, fmt.Sprintf("%d", i))
	}
	joinedVertexIndices := strings.Join(vertexIndices, ", ")
	usd.output.Write([]byte(fmt.Sprintf("        int[] faceVertexIndices = [%s]\n",
		joinedVertexIndices)))

	// TODO: Each mesh will end up having its own material reference from the Materials section
	usd.output.Write([]byte(fmt.Sprintf("        rel material:binding = </Materials/%s>\n", materialID)))

	/**
	 * POINTS
	 */
	pointsComponents := make([]string, 0, positionCount/3)
	for i := 0; i < positionCount; i += 3 {
		pointsComponents = append(pointsComponents, fmt.Sprintf("(%f, %f, %f)",
			currentPositions[i]*PositionScaleConstant,
			currentPositions[i+1]*PositionScaleConstant,
			currentPositions[i+2]*PositionScaleConstant))
	}
	usd.output.Write([]byte(fmt.Sprintf("        point3f[] points = [%s]\n",
		strings.Join(pointsComponents, ", "))))

	/**
	 * NORMALS
	 */
	normalsComponents := make([]string, 0, normalCount/3)
	for i := 0; i < normalCount; i += 3 {
		normalsComponents = append(normalsComponents, fmt.Sprintf("(%f, %f, %f)", currentNormals[i], currentNormals[i+1], currentNormals[i+2]))
	}
	// If this ( is not after the normals, it won't render the file. it is required i guess
	usd.output.Write([]byte(fmt.Sprintf("        normal3f[] primvars:normals = [%s] (\n            interpolation = \"vertex\"\n        )\n", strings.Join(normalsComponents, ", "))))

	/**
	 * NORMAL INDICES
	 */
	normalIndices := make([]string, 0, len(currentNormals))
	for i := 0; i < len(currentNormals)/3; i++ {
		normalIndices = append(normalIndices, fmt.Sprintf("%d", i))
	}
	joinedNormalIndices := strings.Join(normalIndices, ", ")
	usd.output.Write([]byte(fmt.Sprintf("        int[] primvars:normals:indices = [%s]\n",
		joinedNormalIndices)))

	/**
	 * TEXTURE COORDINATES
	 */
	texcoordComponents := make([]string, 0, texcoordCount/2)
	for i := 0; i < texcoordCount; i += 2 {
		texcoordComponents = append(texcoordComponents, fmt.Sprintf("(%f, %f)", currentTexcoords[i], currentTexcoords[i+1]))
	}
	// If this ( is not after the normals, it won't render the file. it is required i guess
	usd.output.Write([]byte(fmt.Sprintf("        float2[] primvars:Texture_uv = [%s] (\n            interpolation = \"faceVarying\"\n        )\n", strings.Join(texcoordComponents, ", "))))

	/**
	 * TEXTURE COORDINATE INDICES
	 */
	texcoordIndices := make([]string, 0, len(currentTexcoords))
	for i := 0; i < len(currentTexcoords)/2; i++ {
		texcoordIndices = append(texcoordIndices, fmt.Sprintf("%d", i))
	}
	joinedTexcoordIndices := strings.Join(normalIndices, ", ")
	usd.output.Write([]byte(fmt.Sprintf("        int[] primvars:Texture_uv:indices = [%s]\n",
		joinedTexcoordIndices)))

	/**
	 * CLOSING MESH
	 */
	//usd.output.Write([]byte("    uniform token subdivisionScheme = \"none\"}\n"))
	usd.output.Write([]byte("    }\n"))

	return nil
}
