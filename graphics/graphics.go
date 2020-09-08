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
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/kpango/glg"
	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/tidwall/gjson"
)

const (
	invertAO = true
	// invertSmoothness is a flag indicating whether the smoothness attribute stored
	// in the gearstack should be inverted into (hopefully?) roughness.
	invertSmoothness = true
	invertMetalness  = false
)

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
	plateMap := platesArray[0].Map()
	plateIndex := int(plateMap["plate_set"].Get("diffuse").Get("plate_index").Int())
	for i := 0; i < newGeomCount; i++ {
		// Use this texture plate for all newly added geometries
		output.plateIndices = append(output.plateIndices, plateIndex)
	}

	err := processTexturePlate("diffuse", plateMap, output)
	if err != nil {
		return err
	}

	err = processTexturePlate("normal", plateMap, output)
	if err != nil {
		return err
	}

	err = processTexturePlate("gearstack", plateMap, output)
	if err != nil {
		return err
	}

	glg.Infof("Position length : %d", len(output.positionVertices))
	glg.Infof("Found plate indices : %+v", output.plateIndices)

	return nil
}

func processTexturePlate(plateName string, texturePlateJSON map[string]gjson.Result, output *processedOutput) error {

	diffuseSet := texturePlateJSON["plate_set"].Get(plateName)
	plateIndex := int(diffuseSet.Get("plate_index").Int())
	plateSizeTemp := diffuseSet.Get("plate_size").Array()
	if len(plateSizeTemp) < 2 {
		return fmt.Errorf("plate size temp has less than two elements for %s plate: Expected 2, Found %d", plateName, len(plateSizeTemp))
	}

	plateSize := [2]int{int(plateSizeTemp[0].Int()), int(plateSizeTemp[1].Int())}

	texturePlacements := diffuseSet.Get("texture_placements").Array()

	// if len(texturePlacements) <= 0 {
	// 	glg.Info("Found 0 texutre placements for this geometry... skipping.")
	// 	//output.plateIndices = append(output.plateIndices, -1)
	// 	return nil
	// }

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

	if len(texturePlacements) <= 0 {
		// No textures to plate, just leave it as a black image
		return nil
	}

	matches, err := filepath.Glob("./output/" + textureTagName + ".*")
	glg.Info("looking for texture with glob ./output/" + textureTagName + ".*")
	if len(matches) > 1 {
		err = errors.New("Found more than one matching texture file name " + textureTagName)
		glg.Error(err)
		return err
	}
	if len(matches) == 0 {
		err = errors.New("Found zero matching texture files matching name " + textureTagName)
		glg.Error(err)
		return err
	}

	// Copy this data into the destination image
	inF, err := os.OpenFile(matches[0], os.O_RDONLY, 0644)
	if err != nil {
		glg.Error(err)
		return err
	}

	img, format, err := image.Decode(inF)
	if err != nil {
		glg.Error(err)
		return err
	}

	var plate *texturePlate
	if plateName == "diffuse" {
		plate = output.texturePlates[plateIndex]
	} else if plateName == "normal" {
		plate = output.normalTexturePlates[plateIndex]
	} else {
		plate = output.gearstackTexturePlates[plateIndex]
	}
	if plate == nil {

		plate = &texturePlate{
			name: textureTagName + "_" + plateName + "." + format,
			size: plateSize,
			data: defaultImageForPlateType(plateName, plateSize),
		}
		if plateName == "diffuse" {
			output.texturePlates[plateIndex] = plate
		} else if plateName == "normal" {
			output.normalTexturePlates[plateIndex] = plate
		} else {
			output.gearstackTexturePlates[plateIndex] = plate
		}
	}

	glg.Debugf("Successfully decoded image with format: %s", format)

	dp := image.Point{posX, posY}
	r := image.Rectangle{dp, dp.Add(image.Point{sizeX, sizeY})}
	draw.Draw(plate.data, r, img, image.ZP, draw.Src)

	return nil
}

func defaultImageForPlateType(plateType string, size [2]int) draw.Image {

	if plateType == "diffuse" {
		return blackImageOfSize(size)
	} else if plateType == "normal" {
		return normalImageOfSize(size)
	} else if plateType == "gearstack" {
		return gearStackImageOfSize(size)
	} else {
		glg.Errorf("Found an uknown plate type for default color: %s", plateType)
		panic("Unsupported texture plate type: " + plateType)
	}
}

func blackImageOfSize(size [2]int) draw.Image {

	img := image.NewRGBA(image.Rect(0, 0, size[0], size[1]))
	black := color.RGBA{0, 0, 0, 255}

	draw.Draw(img, img.Bounds(), image.NewUniform(black), image.ZP, draw.Src)

	return img
}

func normalImageOfSize(size [2]int) draw.Image {
	img := image.NewRGBA(image.Rect(0, 0, size[0], size[1]))
	black := color.RGBA{128, 128, 255, 255}

	draw.Draw(img, img.Bounds(), image.NewUniform(black), image.ZP, draw.Src)

	return img
}

func gearStackImageOfSize(size [2]int) draw.Image {
	img := image.NewRGBA(image.Rect(0, 0, size[0], size[1]))
	clear := color.RGBA{0, 0, 0, 0}

	draw.Draw(img, img.Bounds(), image.NewUniform(clear), image.ZP, draw.Src)

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
			out := make([]int16, 2)
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

func parseVertex32(data []byte, vertexType string, offset int, stride int) [][]float32 {

	result := make([][]float32, 0, 10)
	for i := offset; i < len(data); i += stride {

		if vertexType == "_vertex_format_attribute_short2" {
			out := make([]int16, 2)
			binary.Read(bytes.NewBuffer(data[i:i+4]), binary.LittleEndian, out)
			outFloats := []float32{float32(out[0]), float32(out[1])}
			result = append(result, outFloats)
		} else if vertexType == "_vertex_format_attribute_float2" {
			out := make([]float32, 2)
			binary.Read(bytes.NewBuffer(data[i:i+8]), binary.LittleEndian, out)
			outFloats := []float32{float32(out[0]), float32(out[1])}
			result = append(result, outFloats)
		} else {
			fmt.Println("Found unknown vertex type!!")
		}
	}
	return result
}

// ExplodeGearstack will open the gearstack image at the provided path
// and break it up into the separate physically based rendering (PBR) components.
func ExplodeGearstack(path string) (*PBRTextureCollection, error) {
	inF, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		glg.Errorf("Failed to open the specified file: %s", err.Error())
		return nil, err
	}
	defer inF.Close()

	img, fmt, err := image.Decode(inF)
	if err != nil {
		glg.Errorf("Failed to decode the specified gearstack image: %s", err.Error())
		return nil, err
	}
	glg.Debugf("Decoded image with fmt: %s", fmt)

	pbr, err := ExplodePBRTexture(img)
	if err != nil {
		glg.Errorf("Could not expand image into separate pbr textures: %s", err.Error())
		return nil, err
	}
	pbr.ImageFormat = fmt

	return pbr, nil
}

// ExplodePBRTexture will separate the gearstack provided as an image.Image into the individual
// PBR components (ambient occlusion, metalness, roughness).
func ExplodePBRTexture(img image.Image) (*PBRTextureCollection, error) {

	metalnessImg := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	roughnessImg := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	aoImg := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	emissiveImg := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))

	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(metalnessImg, metalnessImg.Bounds(), image.NewUniform(black), image.ZP, draw.Src)
	draw.Draw(roughnessImg, roughnessImg.Bounds(), image.NewUniform(black), image.ZP, draw.Src)
	draw.Draw(aoImg, aoImg.Bounds(), image.NewUniform(black), image.ZP, draw.Src)
	draw.Draw(emissiveImg, emissiveImg.Bounds(), image.NewUniform(black), image.ZP, draw.Src)

	result := &PBRTextureCollection{
		Metalness:        metalnessImg,
		Roughness:        roughnessImg,
		AmbientOcclusion: aoImg,
		Emissive:         emissiveImg,
	}

	origin := img.Bounds().Min
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	for x := origin.X; x < (origin.X + width); x++ {
		for y := origin.Y; y < (origin.Y + height); y++ {
			ao, smoothness, b, a := img.At(x, y).RGBA()

			// Ambient occlusion
			// I have no clue about this but i think it needs to be inverted to mostly white
			// (that is what the examples show at least)
			finalAO := uint8(ao)
			if invertAO {
				finalAO = 255 - uint8(ao)
			}
			setPixel(result.AmbientOcclusion, finalAO, x, y)

			// Metalness
			// from the slides "In the alpha channel we give metalness 32 values", i take this
			// to mean that metalness is contained in values 0-31 or the lower 5 bits.
			normalized := float64(a&0xFF) / float64(0xFF)
			masked := uint8((normalized * 0xFF)) & 0x1F
			metalness := uint8((float64(masked) / 32.0) * 255.0)
			if invertMetalness {
				metalness = 255 - metalness
			}
			setPixel(result.Metalness, metalness, x, y)

			normalizedEmissive := float64(uint8(b)-40) / (255.0 - 40.0)
			emissive := uint8(255.0 * normalizedEmissive)
			setPixel(result.Emissive, emissive, x, y)

			// Roughness
			originalSmoothness := uint8(smoothness & 0x00FF)
			if invertSmoothness {
				originalSmoothness = 255 - originalSmoothness
			}
			setPixel(result.Roughness, originalSmoothness, x, y)

			//glg.Debugf("ao=0x%x, smoothness=0x%x, maskedMetalness=0x%x, metalness=0x%x", ao, smoothness, masked, metalness)
		}
	}

	return result, nil
}

func setPixel(img draw.Image, val uint8, x, y int) {

	img.Set(x, y, color.RGBA{val, val, val, 255})
}

func writeTextures(processed *processedOutput, pathPrefix string) error {

	for _, plate := range processed.texturePlates {
		writeTexturePlate(plate, pathPrefix)
	}

	for _, plate := range processed.normalTexturePlates {
		writeTexturePlate(plate, pathPrefix)
	}

	for i, plate := range processed.gearstackTexturePlates {
		if plate == nil {
			continue
		}

		// writeTexturePlate(plate)
		pbr, err := ExplodePBRTexture(plate.data)
		if err != nil {
			glg.Errorf("Error trying to expand gearstack texture: %s", err.Error())
			continue
		}

		glg.Infof("Setting pbr texture of index: %d", i)
		processed.pbrTextures[i] = pbr

		aoName := strings.Replace(plate.name, "gearstack", "AO", -1)
		metalnessName := strings.Replace(plate.name, "gearstack", "metalness", -1)
		roughnessName := strings.Replace(plate.name, "gearstack", "roughness", -1)
		emissiveName := strings.Replace(plate.name, "gearstack", "emissive", -1)

		writeTextureFile(pbr.AmbientOcclusion, pathPrefix, aoName)
		writeTextureFile(pbr.Metalness, pathPrefix, metalnessName)
		writeTextureFile(pbr.Roughness, pathPrefix, roughnessName)
		writeTextureFile(pbr.Emissive, pathPrefix, emissiveName)
	}

	return nil
}

func writeTexturePlate(plate *texturePlate, pathPrefix string) error {
	if plate == nil {
		return nil
	}

	return writeTextureFile(plate.data, pathPrefix, plate.name)
}

func writeTextureFile(img draw.Image, pathPrefix, name string) error {

	// Write this to a file now
	filePath := pathPrefix + "/" + name
	format := "jpeg"
	if strings.HasSuffix(name, "png") {
		format = "png"
	}
	glg.Infof("Writing texture file, with format=%s, to: %s", format, filePath)
	outF, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		glg.Error(err)
		return err
	}

	flipped := flipVertically(img)

	if format == "png" {
		err = png.Encode(outF, flipped)
	} else {
		err = jpeg.Encode(outF, flipped, nil)
	}
	if err != nil {
		glg.Error(err)
		return err
	}
	outF.Close()

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
