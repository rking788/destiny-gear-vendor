package bungie

import (
	"encoding/json"
	"strings"

	"github.com/rking788/destiny-gear-vendor/db"
)

func GetAssetDefinition(itemHash uint) (*GearAssetDefinition, error) {
	db, err := db.GetAssetDBConnection()
	if err != nil {
		return nil, err
	}

	definition, err := db.GetAssetDefinition(itemHash)
	if err != nil {
		return nil, err
	}

	assetDefinition := &GearAssetDefinition{}
	decoder := json.NewDecoder(strings.NewReader(definition))
	err = decoder.Decode(assetDefinition)
	if err != nil {
		return nil, err
	}

	return assetDefinition, nil
}
