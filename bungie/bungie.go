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
	assetDefinition.ID = itemHash

	return assetDefinition, nil
}

func GetAllAssetDefinitions() ([]*GearAssetDefinition, error) {

	db, err := db.GetAssetDBConnection()
	if err != nil {
		return nil, err
	}

	definitions, err := db.GetAllAssetDefinitions()
	if err != nil {
		return nil, err
	}

	result := make([]*GearAssetDefinition, 0, len(definitions))
	for _, row := range definitions {
		assetDefinition := &GearAssetDefinition{}
		assetDefinition.ID = row["id"].(uint)

		decoder := json.NewDecoder(strings.NewReader(row["definition"].(string)))
		err = decoder.Decode(assetDefinition)
		if err != nil {
			return nil, err
		}

		result = append(result, assetDefinition)
	}

	return result, nil
}
