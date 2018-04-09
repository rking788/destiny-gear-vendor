package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	ModelPathPrefix       = "./output/gear.scnassets/"
	ModelNamePrefix       = ModelPathPrefix
	TexturePathPrefix     = "./output/"
	LocalGeometryBasePath = "./local_tools/geom/geometry/"
	LocalTextureBasePath  = "./local_tools/geom/textures/"
)

var (
	BungieApiKey = os.Getenv("BUNGIE_API_KEY")

	// Unused coord pair: [1.3330078125, 2.666015625]
	/*"texcoord_offset": [
	    0.401725,
	    0.400094
	  ],
	  "texcoord_scale": [
	    0.396719,
	    0.396719
	  ],*/
	//UnusedX        = 1.3330078125
	//UnusedY        = 2.66015625
	//UnusedX          = 1.333333333333333
	//UnusedY          = 2.666666666666667
)

func main() {

	isCLI := flag.Bool("cli", false, "Use this flag to indicate that the program is being run"+
		" as a CLI program and should print the path to the expected output when done instead"+
		" of running a web server")
	itemHash := flag.Uint("hash", 0, "The item hash from the manifest DB for the asset to use")
	withAllAssets := flag.Bool("all", false, "Use this flag to request that all assets from the manifest be processed")
	withSTL := flag.Bool("stl", false, "Use this to request STL format assets")
	withDAE := flag.Bool("dae", false, "Use this flag to request DAE format assets")
	withGeom := flag.Bool("geom", false, "Indicates that geometries should be parsed and written")
	withTextures := flag.Bool("textures", false, "Indicates that textures should be processed")
	flag.Parse()

	fmt.Printf("IsCLI: %v\n", *isCLI)

	if *isCLI {
		executeCommand(*itemHash, *withAllAssets, *withSTL, *withDAE, *withGeom, *withTextures)
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

func executeCommand(hash uint, withAllAssets, withSTL, withDAE, withGeom, withTextures bool) {
	fmt.Printf("WithSTL: %v\n", withSTL)
	fmt.Printf("WithDAE: %v\n", withDAE)

	if hash == 0 && withAllAssets == false {
		glg.Error("Forgot to provide an item hash!")
		return
	}

	if withSTL == false && withDAE == false {
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
	} else {
		assetDefinition, err := bungie.GetAssetDefinition(hash)
		if err != nil {
			glg.Errorf("Error requesting asset definition from the DB: %s", err.Error())
			return
		}

		assetDefinitions = []*bungie.GearAssetDefinition{assetDefinition}
	}

	for _, assetDefinition := range assetDefinitions {
		glg.Infof("Processing item with hash: %d", assetDefinition.ID)
		if withTextures {
			processTextures(assetDefinition)
		}
		if withGeom {
			processGeometry(assetDefinition, withSTL, withDAE)
		}
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func processGeometry(asset *bungie.GearAssetDefinition, withSTL, withDAE bool) string {

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
			response, _ := client.Do(req)
			if response.StatusCode != 200 {
				glg.Errorf("Failed to download geometry for hash(%d), bad response: %d\n", asset.ID, response.StatusCode)
				return ""
			}

			bodyBytes, _ := ioutil.ReadAll(response.Body)
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

func processTextures(asset *bungie.GearAssetDefinition) {
	for _, textureFile := range asset.Content[0].Textures {
		texturePath := LocalTextureBasePath + textureFile
		if _, err := os.Stat(texturePath); os.IsNotExist(err) {

			glg.Infof("Downloading texture file... %s", textureFile)

			client := http.Client{}
			req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.TexturePrefix+textureFile, nil)
			req.Header.Set("X-API-Key", BungieApiKey)
			response, _ := client.Do(req)

			bodyBytes, _ := ioutil.ReadAll(response.Body)
			ioutil.WriteFile(LocalTextureBasePath+textureFile, bodyBytes, 0644)
		} else {
			glg.Info("Found cached texture file...")
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
	err = ioutil.WriteFile(fmt.Sprintf("./local_tools/%d-%d-%s-meshes.json",
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
