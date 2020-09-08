package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kpango/glg"
	"github.com/rking788/destiny-gear-vendor/bungie"
	"github.com/rking788/destiny-gear-vendor/graphics"
)

/****** TODO *******
- pull geom (.tgx) and texture (.tgx.bin) files out of the item definition for the male index set
*******************/

/**
	"mobileGearCDN": {
	"Geometry": "/common/destiny2_content/geometry/platform/mobile/geometry",
	"Texture": "/common/destiny2_content/geometry/platform/mobile/textures",
	"PlateRegion": "/common/destiny2_content/geometry/platform/mobile/plated_textures",
	"Gear": "/common/destiny2_content/geometry/gear",
	"Shader": "/common/destiny2_content/geometry/platform/mobile/shaders"
**/

const (
	ModelPathPrefix         = "./output/gear.scnassets/"
	ModelNamePrefix         = ModelPathPrefix
	TexturePathPrefix       = "./output/"
	LocalGeometryBasePath   = "./local_tools/geom/geometry/"
	LocalTextureBasePath    = "./local_tools/geom/textures/"
	RenderMeshesBasePath    = "./local_tools/"
	AssetDefinitionBasePath = "./local_tools/"
)

var (
	BungieApiKey = os.Getenv("BUNGIE_API_KEY")
)

func main() {

	isCLI := flag.Bool("cli", false, "Use this flag to indicate that the program is being run"+
		" as a CLI program and should print the path to the expected output when done instead"+
		" of running a web server")
	itemHash := flag.Uint("hash", 0, "The item hash from the manifest DB for the asset to use")
	withAllAssets := flag.Bool("all", false, "Use this flag to request that all assets from the manifest be processed")
	withWeapons := flag.Bool("weapons", false, "Generate models for all weapon assets in the DB")
	withGhosts := flag.Bool("ghosts", false, "Generate models for all ghost assets in the DB")
	withVehicles := flag.Bool("vehicles", false, "Generate models for all vehicle assets in the DB")
	withSTL := flag.Bool("stl", false, "Use this to request STL format assets")
	withDAE := flag.Bool("dae", false, "Use this flag to request DAE format assets")
	withUSDA := flag.Bool("usda", false, "Write the model for the specified Destiny gear in USDZ format")
	withUSDC := flag.Bool("usdc", false, "Write the model for the specified Destiny gear in USDZ format")
	withUSDZ := flag.Bool("usdz", false, "Write the model for the specified Destiny gear in USDZ format")
	withGeom := flag.Bool("geom", false, "Indicates that geometries should be parsed and written")
	withTextures := flag.Bool("textures", false, "Indicates that textures should be processed")
	flag.Parse()

	fmt.Printf("IsCLI: %v\n", *isCLI)

	if *isCLI {
		// TODO: This should combine all this configuration into some kind of
		// struct or something.
		executeCommand(*itemHash, *withAllAssets, *withWeapons, *withGhosts, *withVehicles, *withSTL, *withDAE, *withUSDA, *withUSDC, *withUSDZ, *withGeom, *withTextures)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("Forgot to specify a port")
		return
	}

	// If running in web server mode, setup the routes and start the server
	router := mux.NewRouter()
	router.HandleFunc("/gear-vendor/{hash}/{format}", GetAsset).Methods("GET")

	glg.Error(http.ListenAndServe(":"+port, router))
}

func executeCommand(hash uint, withAllAssets, withWeapons, withGhosts, withVehicles, withSTL, withDAE, withUSDA, withUSDC, withUSDZ, withGeom, withTextures bool) {
	fmt.Printf("WithSTL: %v\n", withSTL)
	fmt.Printf("WithDAE: %v\n", withDAE)
	fmt.Printf("WithUSDA: %v\n", withUSDA)
	fmt.Printf("WithUSDC: %v\n", withUSDC)
	fmt.Printf("WithUSDZ: %v\n", withUSDZ)

	if hash == 0 && withAllAssets == false && withWeapons == false {
		glg.Error("Forgot to provide an item hash!")
		return
	}

	if withSTL == false && withDAE == false && withUSDA == false && withUSDC == false && withUSDZ == false {
		glg.Error("No output format specified!")
		return
	}

	var assetDefinitions []*bungie.GearAssetDefinition
	if withAllAssets {
		var err error
		assetDefinitions, err = bungie.GetAllAssetDefinitions()
		if err != nil {
			glg.Errorf("Error requesting asset definitions for all items: %s", err.Error())
			return
		}
	} else if hash != 0 {
		assetDefinition, err := bungie.GetAssetDefinition(hash)
		if err != nil {
			glg.Errorf("Error requesting asset definition from the DB: %s", err.Error())
			return
		}

		assetDefinitions = []*bungie.GearAssetDefinition{assetDefinition}
	} else {
		assetDefinitions = make([]*bungie.GearAssetDefinition, 0, 20)

		if withWeapons {
			defs, err := bungie.GetWeaponAssetDefinitions()
			if err != nil {
				glg.Errorf("Error requesting weapon asset definitions: %s", err.Error())
				return
			}
			assetDefinitions = append(assetDefinitions, defs...)
		}

		if withGhosts {
			ghosts, err := bungie.GetGhostAssetDefinitions()
			if err != nil {
				glg.Errorf("Error requesting ghost asset definitions: %s", err.Error())
				return
			}
			assetDefinitions = append(assetDefinitions, ghosts...)
		}

		if withVehicles {
			vehicles, err := bungie.GetVehicleAssetDefinitions()
			if err != nil {
				glg.Errorf("Error requesting vehicle asset definitions: %s", err.Error())
				return
			}
			assetDefinitions = append(assetDefinitions, vehicles...)
		}
	}

	for _, assetDefinition := range assetDefinitions {
		glg.Infof("Processing item with hash: %d", assetDefinition.ID)

		// Write the asset definition out to a local file
		writeAssetDefinition(assetDefinition)
		writeGearDescription(assetDefinition)

		if withTextures {
			processTextures(assetDefinition)
		}
		if withGeom {
			processGeometry(assetDefinition, withSTL, withDAE, withUSDA, withUSDC, withUSDZ)
		}
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func writeAssetDefinition(def *bungie.GearAssetDefinition) {
	fullPath := AssetDefinitionBasePath + fmt.Sprintf("%d-asset-def.json", def.ID)
	if fileExists(fullPath) {
		glg.Info("Found cached asset definition")
		return
	}

	outF, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		glg.Errorf("Failed to open asset definition file: %s", err.Error())
	}
	encoder := json.NewEncoder(outF)
	encoder.Encode(def)
	outF.Close()
}

func writeGearDescription(def *bungie.GearAssetDefinition) {
	fullPath := AssetDefinitionBasePath + fmt.Sprintf("%d-gear-%s", def.ID, def.Gear[0])
	if fileExists(fullPath) {
		glg.Info("Found cached gear description")
		return
	}

	glg.Info("Downloading gear description: %s", def.Gear[0])
	client := http.Client{}
	gearURL := bungie.UrlPrefix + bungie.GearPrefix + def.Gear[0]
	fmt.Printf("Requesting gear URL : %s\n", gearURL)
	req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.GearPrefix+def.Gear[0], nil)
	req.Header.Set("X-API-Key", BungieApiKey)
	response, err := client.Do(req)
	if err != nil {
		glg.Errorf("Could not download gear description: %s", err.Error())
		return
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		glg.Errorf("Failed to read gear description response: %s", err.Error())
		return
	}
	defer response.Body.Close()

	err = ioutil.WriteFile(fullPath, responseBytes, 0644)
	if err != nil {
		glg.Errorf("Failed to write asset description locally: %s", err.Error())
		return
	}
}

func processGeometry(asset *bungie.GearAssetDefinition, withSTL, withDAE, withUSDA, withUSDC, withUSDZ bool) string {

	stlOutputPath := fmt.Sprintf("%s/%d.stl", ModelPathPrefix, asset.ID)
	daeOutputPath := fmt.Sprintf("%s/%d.dae", ModelPathPrefix, asset.ID)

	if withDAE && fileExists(daeOutputPath) {
		glg.Infof(fmt.Sprintf("Cached DAE model already exists: %s", daeOutputPath))
		return daeOutputPath
	} else if withSTL && fileExists(stlOutputPath) {
		glg.Infof(fmt.Sprintf("Cached STL model already exists: %s", stlOutputPath))
		return stlOutputPath
	}

	geometries := make([]*bungie.DestinyGeometry, 0, 12)
	if len(asset.Content) < 1 {
		glg.Errorf("*** ERROR *** No content in the asset definition for id(%d) ****\n", asset.ID)
		return ""
	}

	for geomIndex, geometryFile := range asset.Content[0].Geometry {

		geometryPath := LocalGeometryBasePath + geometryFile

		if !fileExists(geometryPath) {
			glg.Info("Downloading geometry file... ")

			client := http.Client{}
			req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.GeometryPrefix+geometryFile, nil)
			req.Header.Set("X-API-Key", BungieApiKey)
			response, err := client.Do(req)

			if response == nil || err != nil {
				glg.Errorf("Failed to request geometryFile: %s error: %s", geometryFile, err.Error())
			}

			if response.StatusCode != 200 {
				glg.Errorf("Failed to download geometry for hash(%d), bad response: %d\n", asset.ID, response.StatusCode)
				return ""
			}

			bodyBytes, _ := ioutil.ReadAll(response.Body)
			response.Body.Close()
			ioutil.WriteFile(geometryPath, bodyBytes, 0644)
		} else {
			glg.Info("Found cached geometry file...")
		}

		glg.Infof("Parsing geometry file... %s", geometryFile)
		geometry := parseGeometryFile(asset, geomIndex, geometryPath)
		geometries = append(geometries, geometry)
	}

	outDir := fmt.Sprintf("%s/%d", ModelPathPrefix, asset.ID)
	if !fileExists(outDir) {
		err := os.Mkdir(outDir, os.ModePerm)
		if err != nil {
			glg.Errorf("Error creating item subdirectory: %s", err.Error())
		}
	}

	if withUSDA || withUSDC || withUSDZ {
		glg.Info("Writing USD model...")
		path := fmt.Sprintf("%s/%d.usda", outDir, asset.ID)
		usdWriter := &graphics.USDWriter{Path: path, TexturePath: outDir}

		err := usdWriter.WriteModel(geometries)
		if err != nil {
			glg.Errorf("Failed to write model for asset = %d: %v", asset.ID, err)
			return ""
		}

		path, err = createUSDZ(outDir, asset.ID, withUSDA, withUSDC, withUSDZ)
		if err != nil {
			glg.Errorf("Error creating USDZ file: %s", err.Error())
			return ""
		}

		return path
	}

	if withDAE {
		glg.Info("Writing DAE model...")
		path := fmt.Sprintf("%s/%d.dae", outDir, asset.ID)
		daeWriter := &graphics.DAEWriter{Path: path, TexturePath: outDir}
		err := daeWriter.WriteModels(geometries)
		if err != nil {
			glg.Errorf("Error trying to write the DAE model file!!: %s", err.Error())
			return ""
		}

		return path
	}

	if withSTL {
		glg.Info("Writing STL model...")
		path := fmt.Sprintf("%s/%d.stl", outDir, asset.ID)
		stlWriter := &graphics.STLWriter{Path: path}
		err := stlWriter.WriteModels(geometries)
		if err != nil {
			glg.Errorf("Error trying to write the STL model file!!: %s", err.Error())
			return ""
		}

		return path
	}

	return ""
}

func convertASCIIToBinary(path string) error {

	usdcPath := strings.Replace(path, "usda", "usdc", -1)

	err := exec.Command("usdcat", "-o", usdcPath, path).Run()

	return err
}

func zipUSDZ(dir string, id uint, texturePaths []string, outPath string) error {

	// the converted USDC model
	usdc := fmt.Sprintf("%s/%d.usdc", dir, id)

	included := make([]string, 0, 15)
	included = append(included, usdc)
	included = append(included, texturePaths...)

	usdzFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer usdzFile.Close()

	zipWriter := zip.NewWriter(usdzFile)
	if err != nil {
		return err
	}
	defer zipWriter.Close()

	for i := range included {

		zipfile, err := os.Open(included[i])
		if err != nil {
			return err
		}
		defer zipfile.Close()

		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// I have no idea why (and it is outlined in the docs that you should not edit this after CreateHeader) but
		// clearing these flags is required to generate a valid USDZ file. I tried working around this with a temp
		// buffer so the size would ideally would be known when it is copied but that didn't help.
		// Not sure why but here is what the Wiki page says for it:
		// "If the bit at offset 3 (0x08) of the general-purpose flags field is set, then the CRC-32 and file sizes
		// are not known when the header is written. The fields in the local header are filled with zero, and the
		// CRC-32 and size are appended in a 12-byte structure (optionally preceded by a 4-byte signature) immediately
		// after the compressed data:"
		// Find out more here: https://en.wikipedia.org/wiki/Zip_(file_format)
		header.Flags = 0

		if _, err = io.Copy(writer, zipfile); err != nil {
			return err
		}
	}

	return nil
}

func alignUSDZ(unalignedUSDZPath, alignedUSDZPath string) error {

	err := exec.Command("zipalign", "-f", "64", unalignedUSDZPath, alignedUSDZPath).Run()

	return err
}

func createUSDZ(dir string, id uint, withUSDA, withUSDC, withUSDZ bool) (string, error) {

	usdaPath := fmt.Sprintf("%s/%d.usda", dir, id)
	err := convertASCIIToBinary(usdaPath)
	if err != nil {
		return "", err
	}
	if !withUSDA {
		defer os.Remove(usdaPath)
	}

	// .jpeg and .jpg textures
	texturePaths, err := filepath.Glob(fmt.Sprintf("%s/*.j*g", dir))
	if err != nil {
		return "", err
	}

	// .png textures
	pngs, err := filepath.Glob(fmt.Sprintf("%s/*.png", dir))
	if err != nil {
		return "", err
	}
	texturePaths = append(texturePaths, pngs...)
	if !withUSDA && !withUSDC {
		// Cleanup textures if they will not be used after they are zipped up
		defer func(paths []string) {
			for _, f := range paths {
				os.Remove(f)
			}
		}(texturePaths)
	}

	unalignedUSDZPath := fmt.Sprintf("%s/%d-unaligned.usdz", dir, id)
	alignedUSDZPath := fmt.Sprintf("%s/%d.usdz", dir, id)
	err = zipUSDZ(dir, id, texturePaths, unalignedUSDZPath)
	if err != nil {
		return "", err
	}
	if !withUSDC {
		defer os.Remove(strings.Replace(usdaPath, "usda", "usdc", -1))
	}

	err = alignUSDZ(unalignedUSDZPath, alignedUSDZPath)
	if err != nil {
		return "", err
	}
	os.Remove(unalignedUSDZPath)

	return alignedUSDZPath, nil
}

func processTextures(asset *bungie.GearAssetDefinition) {
	if len(asset.Content) <= 0 {
		return
	}

	for _, textureFile := range asset.Content[0].Textures {
		texturePath := LocalTextureBasePath + textureFile
		if _, err := os.Stat(texturePath); os.IsNotExist(err) {

			glg.Infof("Downloading texture file... %s", textureFile)

			client := http.Client{}
			req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.TexturePrefix+textureFile, nil)
			req.Header.Set("X-API-Key", BungieApiKey)
			response, err := client.Do(req)

			if response == nil || err != nil {
				glg.Errorf("Error downloading textureFile: %s error: %s", textureFile, err.Error())
				continue
			}

			if response.StatusCode != 200 {
				continue
			}

			bodyBytes, _ := ioutil.ReadAll(response.Body)
			response.Body.Close()
			ioutil.WriteFile(LocalTextureBasePath+textureFile, bodyBytes, 0644)
		} else {
			glg.Infof("Found cached texture file... %s", textureFile)
		}

		destinyTexture := parseTextureFile(texturePath)

		glg.Infof("Parsed texture: %+v", destinyTexture)

		// Write all images to disk after parsing
		for _, file := range destinyTexture.Files {

			textureOutputPath := TexturePathPrefix + file.Name + file.Extension
			if _, err := os.Stat(textureOutputPath); os.IsNotExist(err) {
				ioutil.WriteFile(textureOutputPath, file.Data, 0644)
			} else {
				glg.Info("Cached texture file found")
			}
		}
	}
}

func parseGeometryFile(asset *bungie.GearAssetDefinition, index int, path string) *bungie.DestinyGeometry {

	f, err := os.Open(path)
	if err != nil {
		glg.Errorf("Failed to open geometry file with error: %s", err.Error())
		return nil
	}
	defer f.Close()

	geom := &bungie.DestinyGeometry{}
	geom.Files = make([]*bungie.GeometryFile, 0)

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
		file := &bungie.GeometryFile{}

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

		glg.Debugf("Finished reading file metadata: %+v", file)
	}

	for _, file := range geom.Files {
		f.Seek(file.StartAddr, 0)

		file.Data = make([]byte, file.Length)
		f.Read(file.Data)

		if file.Name == "render_metadata.js" {
			glg.Debugf("Found render_metadata.js file!!")
			geom.MeshesBytes = file.Data
		}
	}

	safeGeomName := strings.Replace(filepath.Base(path), ".", "", -1)
	err = ioutil.WriteFile(fmt.Sprintf(RenderMeshesBasePath+"%d-%d-%s-meshes.json",
		asset.ID, index, safeGeomName), geom.MeshesBytes, 0644)
	if err != nil {
		glg.Errorf("Failed to write render meshes: %s", err.Error())
	}

	return geom
}

func parseTextureFile(path string) *bungie.DestinyTexture {

	f, err := os.Open(path)
	if err != nil {
		glg.Errorf("Failed to open geometry file with error: %s", err.Error())
		return nil
	}
	defer f.Close()

	text := &bungie.DestinyTexture{}
	text.Files = make([]*bungie.TextureFile, 0)

	// Read file metadata
	buf := make([]byte, 272)
	f.Read(buf)
	metaBuffer := bytes.NewBuffer(buf)

	/** FORMAT:
	** - Extension (4 bytes)
	** - Version (4 bytes)
	** - HeaderSize (4 bytes)
	** - FileCount (4 bytes)
	** - FileIdentifier (256 bytes)
	** - N Files with the following format:
	** -- Name (256 bytes)
	** -- Offset (8 bytes)
	** -- Size (8 bytes)
	** -- Data ("Size" bytes)
	** - FileData[FileCount] (n bytes)
	**/
	extension := make([]byte, 4)
	binary.Read(metaBuffer, binary.LittleEndian, extension)

	text.Extension = string(extension)

	// Unknown
	metaBuffer.Next(4)

	binary.Read(metaBuffer, binary.LittleEndian, &text.HeaderSize)
	binary.Read(metaBuffer, binary.LittleEndian, &text.FileCount)
	nameBuf := make([]byte, 256)
	binary.Read(metaBuffer, binary.LittleEndian, &nameBuf)

	endOfName := bytes.IndexByte(nameBuf, 0)
	if endOfName == -1 {
		glg.Warnf("Failed to find the null byte in filename: %s", string(nameBuf))
		endOfName = len(nameBuf)
	}
	text.Name = string(nameBuf[:endOfName])

	// Read each of the individual files
	for i := int32(0); i < text.FileCount; i++ {
		file := &bungie.TextureFile{}

		nameBuf := make([]byte, 256)
		f.Read(nameBuf)
		n := bytes.IndexByte(nameBuf, 0)

		file.Name = string(nameBuf[:n])

		startAddrBuf := make([]byte, 8)
		f.Read(startAddrBuf)
		binary.Read(bytes.NewBuffer(startAddrBuf), binary.LittleEndian, &file.Offset)

		lengthBuffer := make([]byte, 8)
		f.Read(lengthBuffer)
		binary.Read(bytes.NewBuffer(lengthBuffer), binary.LittleEndian, &file.Size)

		text.Files = append(text.Files, file)

		glg.Debugf("Finished reading file metadata: %+v\n", file)
	}

	for _, file := range text.Files {
		f.Seek(file.Offset, 0)

		file.Data = make([]byte, file.Size)
		f.Read(file.Data)

		if file.Data[0] == 0x89 &&
			file.Data[1] == 0x50 &&
			file.Data[2] == 0x4E &&
			file.Data[3] == 0x47 {
			file.Extension = ".png"
		} else if file.Data[0] == 0xFF &&
			file.Data[1] == 0xD8 {
			file.Extension = ".jpg"
		} else {
			glg.Error("Unknown texture image file format!")
		}
	}

	// ioutil.WriteFile("./local_tools/"+ModelNamePrefix+"-meshes.json", geom.MeshesBytes, 0644)

	return text
}
