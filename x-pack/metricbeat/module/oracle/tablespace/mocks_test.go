package tablespace

import (
	"database/sql"
	"github.com/pkg/errors"
)

type happyMockExtractor struct {
	happyDataFiles
	happyFreeSpaceData
	happyTempFreeSpaceData
}

type errorDataFilesMockExtractor struct {
	errorDataFiles
	happyFreeSpaceData
	happyTempFreeSpaceData
}

type errorFreeSpaceDataMockExtractor struct {
	happyDataFiles
	errorFreeSpaceData
	happyTempFreeSpaceData
}

type errorTempFreeSpaceDataMockExtractor struct {
	happyDataFiles
	happyFreeSpaceData
	errorTempFreeSpaceData
}

type errorFreeSpaceData struct{}
func (errorFreeSpaceData) freeSpaceData() ([]freeSpace, error) {
	return nil, errors.New("data files error")
}

type errorTempFreeSpaceData struct{}
func (errorTempFreeSpaceData) tempFreeSpaceData() ([]tempFreeSpace, error) {
	return nil, errors.New("data files error")
}

type errorDataFiles struct{}
func (errorDataFiles) dataFilesData() ([]dataFile, error) {
	return nil, errors.New("data files error")
}

type happyDataFiles struct{}

func (h happyDataFiles) dataFilesData() ([]dataFile, error) {
	return []dataFile{
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux01.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 18, Valid: true}, TablespaceName: sql.NullString{String: "SYSAUX", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999994}, AvailableForUserBytes: sql.NullInt64{Int64: 99999994, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 9999990}},
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux02.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 181, Valid: true}, TablespaceName: sql.NullString{String: "SYSAUX", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999995}, AvailableForUserBytes: sql.NullInt64{Int64: 99999995, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 9999991}},
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux03.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 182, Valid: true}, TablespaceName: sql.NullString{String: "SYSAUX", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999996}, AvailableForUserBytes: sql.NullInt64{Int64: 99999996, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 9999992}},

		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/system01.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 18, Valid: true}, TablespaceName: sql.NullString{String: "SYSTEM", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999994}, AvailableForUserBytes: sql.NullInt64{Int64: 9999994, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 999990}},
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/temp012017-03-02_07-54-38-075-AM.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 18, Valid: true}, TablespaceName: sql.NullString{String: "TEMP", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999994}, AvailableForUserBytes: sql.NullInt64{Int64: 9999994, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 999991}},
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/undotbs01.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 18, Valid: true}, TablespaceName: sql.NullString{String: "UNDOTBS1", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999994}, AvailableForUserBytes: sql.NullInt64{Int64: 9999994, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 999992}},
		{FileName: sql.NullString{String: "/u02/app/oracle/oradata/ORCLCDB/orclpdb1/users01.dbf", Valid: true}, FileID: sql.NullInt64{Int64: 18, Valid: true}, TablespaceName: sql.NullString{String: "USERS", Valid: true}, Status: sql.NullString{String: "AVAILABLE", Valid: true}, MaxFileSizeBytes: sql.NullInt64{Valid: true, Int64: 9999994}, AvailableForUserBytes: sql.NullInt64{Int64: 9999994, Valid: true}, OnlineStatus: sql.NullString{String: "ONLINE", Valid: true}, TotalSizeBytes: sql.NullInt64{Valid: true, Int64: 999993}},
	}, nil
}

type happyTempFreeSpaceData struct{}

func (happyTempFreeSpaceData) tempFreeSpaceData() ([]tempFreeSpace, error) {
    return []tempFreeSpace{{TablespaceName: "TEMP", TablespaceSize: sql.NullInt64{Valid: true, Int64: 99999}, AllocatedSpace: sql.NullInt64{Valid: true, Int64: 99999}, FreeSpace: sql.NullInt64{Int64: 99999, Valid: true}}}, nil
}

type happyFreeSpaceData struct{}

func (happyFreeSpaceData) freeSpaceData() ([]freeSpace, error) {
    return []freeSpace{
        {TablespaceName: "SYSTEM", TotalBytes: sql.NullInt64{Int64: 9990, Valid: true}},
        {TablespaceName: "SYSAUX", TotalBytes: sql.NullInt64{Int64: 9999, Valid: true}},
        {TablespaceName: "UNDOTBS1", TotalBytes: sql.NullInt64{Int64: 9999, Valid: true}},
        {TablespaceName: "USERS", TotalBytes: sql.NullInt64{Int64: 9999, Valid: true}},
    }, nil
}
