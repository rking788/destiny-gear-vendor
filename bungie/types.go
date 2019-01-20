package bungie

import (
	"fmt"
)

const (
	UrlPrefix          = "http://www.bungie.net"
	GeometryPrefix     = "/common/destiny2_content/geometry/platform/mobile/geometry/"
	TexturePrefix      = "/common/destiny2_content/geometry/platform/mobile/textures/"
	GearPrefix         = "/common/destiny2_content/geometry/gear/"
	PlatedRegionPrefix = "/common/destiny2_content/geometry/platform/mobile/plated_textures/"
	ShaderPrefix       = "/common/destiny2_content/geometry/platform/mobile/shaders/"
)

type DestinyGeometry struct {
	Extension   string
	HeaderSize  int32
	FileCount   int32
	Name        string
	MeshesBytes []byte
	Files       []*GeometryFile
}

func (geom *DestinyGeometry) GetFileByName(name string) *GeometryFile {

	for _, file := range geom.Files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

type GeometryFile struct {
	Name      string
	StartAddr int64
	Length    int64
	Data      []byte
}

type DestinyTexture struct {
	Extension  string
	HeaderSize int32
	FileCount  int32
	Name       string
	Files      []*TextureFile
}

func (t *DestinyTexture) String() string {
	return fmt.Sprintf("%+v", *t)
}

type TextureFile struct {
	Name      string
	Extension string
	Offset    int64
	Size      int64
	Data      []byte
}

func (f *TextureFile) String() string {
	return fmt.Sprintf("{Name:%s Extension:%s Offset:%d Size:%d Data:[%d]",
		f.Name, f.Extension, f.Offset, f.Size, len(f.Data))
}

type GearAssetDefinition struct {
	ID      uint
	Gear    []string       `json:"gear"`
	Content []*GearContent `json:"content"`
}

type GearContent struct {
	Platform        string                 `json:"platform"`
	Geometry        []string               `json:"geometry"`
	Textures        []string               `json:"textures"`
	DyeIndexSet     *IndexSet              `json:"dye_index_set"`
	RegionIndexSets map[string][]*IndexSet `json:"region_index_sets"`
}

func (g *GearContent) String() string {
	return fmt.Sprintf("%+v", *g)
}

type IndexSet struct {
	Textures []int `json:"textures"`
	Geometry []int `json:"geometry"`
}

func (is *IndexSet) String() string {
	return fmt.Sprintf("%+v", *is)
}
