package db

import (
	"fmt"
	"os"

	"database/sql"

	"github.com/kpango/glg"
	_ "github.com/lib/pq" // Only want to import the interface here
)

// AssetDB represents the database containing all of the asset definitions
type AssetDB struct {
	Database *sql.DB
}

var assetDB *AssetDB

// initAssetDatabase is in charge of preparing any Statements that will be commonly used as well
// as setting up the database connection pool.
func initAssetDatabase() error {

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println("DB errror: ", err.Error())
		return err
	}

	assetDB = &AssetDB{
		Database: db,
	}

	return nil
}

// GetAssetDBConnection is a helper for getting a connection to the DB based on
// environment variables or some other method.
func GetAssetDBConnection() (*AssetDB, error) {

	if assetDB == nil {
		fmt.Println("Initializing db!")
		err := initAssetDatabase()
		if err != nil {
			fmt.Println("Failed to initialize the database: ", err.Error())
			return nil, err
		}
	}

	return assetDB, nil
}

func (db *AssetDB) GetAssetDefinition(id uint) (string, error) {

	json := ""
	err := db.Database.QueryRow("SELECT json FROM assets where id = $1", id).Scan(&json)
	if err != nil {
		return "", err
	}

	return json, nil
}

func (db *AssetDB) GetAllAssetDefinitions() ([]map[string]interface{}, error) {

	result := make([]map[string]interface{}, 0, 200)
	rows, err := db.Database.Query("SELECT id, json FROM assets")
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var json string
		var id uint
		rows.Scan(&id, &json)
		entry := make(map[string]interface{})
		entry["id"] = id
		entry["definition"] = json
		result = append(result, entry)
	}

	return result, nil
}

/**
 * 1498876634	- kinetic
 * 2465295065 	- energy
 * 953998645 	- power
 * item_type_name == 'weapon ornament'
 */
func (db *AssetDB) GetWeaponAssetDefinitions() ([]map[string]interface{}, error) {

	result := make([]map[string]interface{}, 0, 200)
	rows, err := db.Database.Query("SELECT assets.id, assets.json FROM assets, items where " +
		"assets.id = items.item_hash AND (items.bucket_type_hash IN (953998645, 2465295065, " +
		"1498876634)) OR (item_type_name = 'weapon ornament'))")

	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var json string
		var id uint
		rows.Scan(&id, &json)
		entry := make(map[string]interface{})
		entry["id"] = id
		entry["definition"] = json
		result = append(result, entry)
	}

	glg.Debugf("Found %d weapon asset definitions", len(result))
	return result, nil
}

/**
 * 	Ghosts	- 4023194814
 */
func (db *AssetDB) GetGhostDefinitions() ([]map[string]interface{}, error) {

	result := make([]map[string]interface{}, 0, 200)
	rows, err := db.Database.Query("SELECT assets.id, assets.json FROM assets, items where " +
		"assets.id = items.item_hash AND items.bucket_type_hash = 4023194814")

	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var json string
		var id uint
		rows.Scan(&id, &json)
		entry := make(map[string]interface{})
		entry["id"] = id
		entry["definition"] = json
		result = append(result, entry)
	}

	glg.Debugf("Found %d ghost asset definitions", len(result))
	return result, nil
}

/**
 * 	Ships		- 284967655
 *	Vehicles	- 2025709351
 */
func (db *AssetDB) GetVehicleDefinitions() ([]map[string]interface{}, error) {

	result := make([]map[string]interface{}, 0, 200)
	rows, err := db.Database.Query("SELECT assets.id, assets.json FROM assets, items where " +
		"assets.id = items.item_hash AND items.bucket_type_hash IN (284967655, 2025709351)")

	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var json string
		var id uint
		rows.Scan(&id, &json)
		entry := make(map[string]interface{})
		entry["id"] = id
		entry["definition"] = json
		result = append(result, entry)
	}

	glg.Debugf("Found %d vehicle asset definitions", len(result))
	return result, nil
}

// Item contains the information needed to display a list of all items including
// a URL to a thumbnail for that item
type Item struct {
	Hash string
	Name string
	Icon string
	Tier int
}

// GetIconLookup will select all item entries including the hash, name, and icon to be
// used to create a simple list of items and their thumbnails. The result is a map keyed by
// the item hash.
func (db *AssetDB) GetIconLookup() (map[string]*Item, error) {

	result := make(map[string]*Item)
	rows, err := db.Database.Query("SELECT item_hash, item_name, icon, tier_type FROM items")

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		item := &Item{}
		rows.Scan(&item.Hash, &item.Name, &item.Icon, &item.Tier)

		result[item.Hash] = item
	}

	return result, nil
}
