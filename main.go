package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"

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
	ModelPathPrefix       = "./output/"
	ModelNamePrefix       = ModelPathPrefix + "better-devils"
	TexturePathPrefix     = "./output/"
	LocalGeometryBasePath = "./local_tools/geom/geometry/"
	LocalTextureBasePath  = "./local_tools/geom/textures/"
)

var (
	BungieApiKey = os.Getenv("BUNGIE_API_KEY")

	//LastWordGeometries = [5]string{"8458a82dec5290cdbc18fa568b94ff99.tgxm", "5bb9e8681f0423e7d89a1febe42457ec.tgxm", "cf97cbfcaae5736094c320b9e3378aa2.tgxm", "f878c2e86541fbf165747362eb3d54fc.tgxm", "4a00ec1e50813252fb0b1341adf1b675.tgxm"}

	// SunshotGeometries = []string{"21b966d2b3e9338b49b5243ecbdcccca.tgxm", "57152585e9f5300a5475478c8ea1f448.tgxm", "60fc8d77e90b35adddf0a99a99facf35.tgxm", "ae800a88325ed4b9e7bc32c86182ae75.tgxm", "defdbb6dcbce8fcd85422f44e53bc4c2.tgxm"}

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
	withSTL := flag.Bool("stl", false, "Use this to request STL format assets")
	withDAE := flag.Bool("dae", false, "Use this flag to request DAE format assets")
	withGeom := flag.Bool("geom", false, "Indicates that geometries should be parsed and written")
	withTextures := flag.Bool("textures", false, "Indicates that textures should be processed")
	flag.Parse()

	fmt.Printf("IsCLI: %v\n", *isCLI)

	if *isCLI {
		executeCommand(*itemHash, *withSTL, *withDAE, *withGeom, *withTextures)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("Forgot to specify a port")
		return
	}

	// If running in web server mode, setup the routes and start the server
	router := mux.NewRouter()
	router.HandleFunc("/gear/{hash}/{format}", GetAsset).Methods("GET")

	fmt.Println(http.ListenAndServe(":"+port, router))
}

func executeCommand(hash uint, withSTL, withDAE, withGeom, withTextures bool) {
	fmt.Printf("WithSTL: %v\n", withSTL)
	fmt.Printf("WithDAE: %v\n", withDAE)

	if hash == 0 {
		fmt.Println("Forgot to provide an item hash!")
		return
	}

	if withSTL == false && withDAE == false {
		fmt.Println("No output format specified!")
		return
	}

	assetDefinition, err := bungie.GetAssetDefinition(hash)
	if err != nil {
		fmt.Printf("Error requesting asset definition from the DB: %s\n", err.Error())
		return
	}

	fmt.Printf("Ready to go with this item def: %+v\n", assetDefinition)

	if withGeom {
		processGeometry(assetDefinition, withSTL, withDAE)
	}
	if withTextures {
		processTextures(assetDefinition)
	}
}

func processGeometry(asset *bungie.GearAssetDefinition, withSTL, withDAE bool) string {

	geometries := make([]*bungie.DestinyGeometry, 0, 12)
	for _, geometryFile := range asset.Content[0].Geometry {

		geometryPath := LocalGeometryBasePath + geometryFile

		if _, err := os.Stat(geometryPath); os.IsNotExist(err) {
			fmt.Println("Downloading geometry file... ", geometryFile)

			client := http.Client{}
			req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.GeometryPrefix+geometryFile, nil)
			req.Header.Set("X-API-Key", BungieApiKey)
			response, _ := client.Do(req)

			bodyBytes, _ := ioutil.ReadAll(response.Body)
			ioutil.WriteFile(geometryPath, bodyBytes, 0644)
		} else {
			fmt.Println("Found cached geometry file...")
		}

		fmt.Println("Parsing geometry file... ")
		geometry := parseGeometryFile(geometryPath)
		geometries = append(geometries, geometry)
	}

	if withSTL {
		fmt.Println("Writing STL model...")
		path := fmt.Sprintf(ModelNamePrefix + "0-auto.stl")
		stlWriter := &graphics.STLWriter{path}
		err := stlWriter.WriteModels(geometries)
		if err != nil {
			fmt.Println("Error trying to write the STL model file!!: ", err.Error())
			return ""
		}

		return path
	}

	if withDAE {
		fmt.Println("Writing DAE model...")
		path := fmt.Sprintf(ModelNamePrefix + "0-auto.dae")
		daeWriter := &graphics.DAEWriter{path}
		err := daeWriter.WriteModels(geometries)
		if err != nil {
			fmt.Println("Error trying to write the DAE model file!!: ", err.Error())
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

			fmt.Println("Downloading texture file... ", textureFile)

			client := http.Client{}
			req, _ := http.NewRequest("GET", bungie.UrlPrefix+bungie.TexturePrefix+textureFile, nil)
			req.Header.Set("X-API-Key", BungieApiKey)
			response, _ := client.Do(req)

			bodyBytes, _ := ioutil.ReadAll(response.Body)
			ioutil.WriteFile(LocalTextureBasePath+textureFile, bodyBytes, 0644)
		} else {
			fmt.Println("Found cached texture file...")
		}

		destinyTexture := parseTextureFile(texturePath)

		fmt.Printf("Parsed texture: %+v\n", destinyTexture)

		// Write all images to disk after parsing
		for _, file := range destinyTexture.Files {

			textureOutputPath := TexturePathPrefix + file.Name + file.Extension
			if _, err := os.Stat(textureOutputPath); os.IsNotExist(err) {
				ioutil.WriteFile(textureOutputPath, file.Data, 0644)
			} else {
				fmt.Println("Cached texture file found")
			}
		}
	}
}

func parseGeometryFile(path string) *bungie.DestinyGeometry {

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Failed to open geometry file with error: ", err.Error())
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

	ioutil.WriteFile("./local_tools/"+ModelNamePrefix+"-meshes.json", geom.MeshesBytes, 0644)

	return geom
}

func parseTextureFile(path string) *bungie.DestinyTexture {

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Failed to open geometry file with error: ", err.Error())
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

		//		fmt.Printf("Finished reading file metadata: %+v\n", file)
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
			fmt.Println("ERROR: Unknown texture image file format!")
		}
	}

	// ioutil.WriteFile("./local_tools/"+ModelNamePrefix+"-meshes.json", geom.MeshesBytes, 0644)

	return text
}
