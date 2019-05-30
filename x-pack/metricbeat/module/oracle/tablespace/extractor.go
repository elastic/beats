package tablespace

import "database/sql"

type tablespaceExtractMethods interface {
	dataFilesData() ([]dataFile, error)
	tempFreeSpaceData() ([]tempFreeSpace, error)
	freeSpaceData() ([]freeSpace, error)
}

type extractedData struct {
	dataFiles     []dataFile
	freeSpace     []freeSpace
	tempFreeSpace []tempFreeSpace
}

type tablespaceExtractor struct {
	db *sql.DB
}
