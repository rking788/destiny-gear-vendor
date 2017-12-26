package db

import (
	"fmt"
	"os"

	"database/sql"

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
