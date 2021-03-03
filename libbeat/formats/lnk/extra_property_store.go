// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package lnk

import (
	"encoding/binary"
	"errors"
	"math"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

const (
	vtEmpty           uint32 = 0x0000
	vtNull            uint32 = 0x0001
	vtI2              uint32 = 0x0002
	vtI4              uint32 = 0x0003
	vtR4              uint32 = 0x0004
	vtR8              uint32 = 0x0005
	vtCY              uint32 = 0x0006
	vtDate            uint32 = 0x0007
	vtBStr            uint32 = 0x0008
	vtError           uint32 = 0x000A
	vtBool            uint32 = 0x000B
	vtDecimal         uint32 = 0x000E
	vtI1              uint32 = 0x0010
	vtUI1             uint32 = 0x0011
	vtUI2             uint32 = 0x0012
	vtUI4             uint32 = 0x0013
	vtI8              uint32 = 0x0014
	vtUI8             uint32 = 0x0015
	vtInt             uint32 = 0x0016
	vtUInt            uint32 = 0x0017
	vtLPStr           uint32 = 0x001E
	vtLPWStr          uint32 = 0x001F
	vtFiletime        uint32 = 0x0040
	vtBlob            uint32 = 0x0041
	vtStream          uint32 = 0x0042
	vtStorage         uint32 = 0x0043
	vtStreamedObject  uint32 = 0x0044
	vtStoredObject    uint32 = 0x0045
	vtBlobObject      uint32 = 0x0046
	vtCF              uint32 = 0x0047
	vtCLSID           uint32 = 0x0048
	vtVersionedStream uint32 = 0x0049
	// vectors
	vtVectorI2       uint32 = 0x1002
	vtVectorI4       uint32 = 0x1003
	vtVectorR4       uint32 = 0x1004
	vtVectorR8       uint32 = 0x1005
	vtVectorCY       uint32 = 0x1006
	vtVectorDate     uint32 = 0x1007
	vtVectorBStr     uint32 = 0x1008
	vtVectorError    uint32 = 0x100A
	vtVectorBool     uint32 = 0x100B
	vtVectorVariant  uint32 = 0x100C
	vtVectorI1       uint32 = 0x1010
	vtVectorUI1      uint32 = 0x1011
	vtVectorUI2      uint32 = 0x1012
	vtVectorUI4      uint32 = 0x1013
	vtVectorI8       uint32 = 0x1014
	vtVectorUI8      uint32 = 0x1015
	vtVectorLPStr    uint32 = 0x101E
	vtVectorLPWStr   uint32 = 0x101F
	vtVectorFiletime uint32 = 0x1040
	vtVectorCF       uint32 = 0x1047
	vtVectorCLSID    uint32 = 0x1048
	// arrays
	vtArrayI2      uint32 = 0x2002
	vtArrayI4      uint32 = 0x2003
	vtArrayR4      uint32 = 0x2004
	vtArrayR8      uint32 = 0x2005
	vtArrayCY      uint32 = 0x2006
	vtArrayDate    uint32 = 0x2007
	vtArrayBStr    uint32 = 0x2008
	vtArrayError   uint32 = 0x200A
	vtArrayBool    uint32 = 0x200B
	vtArrayVariant uint32 = 0x200C
	vtArrayDecimal uint32 = 0x200E
	vtArrayI1      uint32 = 0x2010
	vtArrayUI1     uint32 = 0x2011
	vtArrayUI2     uint32 = 0x2012
	vtArrayUI4     uint32 = 0x2013
	vtArrayInt     uint32 = 0x2016
	vtArrayUint    uint32 = 0x2017
)

var (
	propertyTypes = map[uint32]string{
		vtEmpty:           "VT_EMPTY",
		vtNull:            "VT_NULL",
		vtI2:              "VT_I2",
		vtI4:              "VT_I4",
		vtR4:              "VT_R4",
		vtR8:              "VT_R8",
		vtCY:              "VT_CY",
		vtDate:            "VT_DATE",
		vtBStr:            "VT_BSTR",
		vtError:           "VT_ERROR",
		vtBool:            "VT_BOOL",
		vtDecimal:         "VT_DECIMAL",
		vtI1:              "VT_I1",
		vtUI1:             "VT_UI1",
		vtUI2:             "VT_UI2",
		vtUI4:             "VT_UI4",
		vtI8:              "VT_I8",
		vtUI8:             "VT_UI8",
		vtInt:             "VT_INT",
		vtUInt:            "VT_UINT",
		vtLPStr:           "VT_LPSTR",
		vtLPWStr:          "VT_LPWSTR",
		vtFiletime:        "VT_FILETIME",
		vtBlob:            "VT_BLOB",
		vtStream:          "VT_STREAM",
		vtStorage:         "VT_STORAGE",
		vtStreamedObject:  "VT_STREAMED_OBJECT",
		vtStoredObject:    "VT_STORED_OBJECT",
		vtBlobObject:      "VT_BLOB_OBJECT",
		vtCF:              "VT_CF",
		vtCLSID:           "VT_CLSID",
		vtVersionedStream: "VT_VERSIONED_STREAM",
		vtVectorI2:        "VT_VECTOR | VT_I2",
		vtVectorI4:        "VT_VECTOR | VT_I4",
		vtVectorR4:        "VT_VECTOR | VT_R4",
		vtVectorR8:        "VT_VECTOR | VT_R8",
		vtVectorCY:        "VT_VECTOR | VT_CY",
		vtVectorDate:      "VT_VECTOR | VT_DATE",
		vtVectorBStr:      "VT_VECTOR | VT_BSTR",
		vtVectorError:     "VT_VECTOR | VT_ERROR",
		vtVectorBool:      "VT_VECTOR | VT_BOOL",
		vtVectorVariant:   "VT_VECTOR | VT_VARIANT",
		vtVectorI1:        "VT_VECTOR | VT_I1",
		vtVectorUI1:       "VT_VECTOR | VT_UI1",
		vtVectorUI2:       "VT_VECTOR | VT_UI2",
		vtVectorUI4:       "VT_VECTOR | VT_UI4",
		vtVectorI8:        "VT_VECTOR | VT_I8",
		vtVectorUI8:       "VT_VECTOR | VT_UI8",
		vtVectorLPStr:     "VT_VECTOR | VT_LPSTR",
		vtVectorLPWStr:    "VT_VECTOR | VT_LPWSTR",
		vtVectorFiletime:  "VT_VECTOR | VT_FILETIME",
		vtVectorCF:        "VT_VECTOR | VT_CF",
		vtVectorCLSID:     "VT_VECTOR | VT_CLSID",
		vtArrayI2:         "VT_ARRAY | VT_I2",
		vtArrayI4:         "VT_ARRAY | VT_I4",
		vtArrayR4:         "VT_ARRAY | VT_R4",
		vtArrayR8:         "VT_ARRAY | VT_R8",
		vtArrayCY:         "VT_ARRAY | VT_CY",
		vtArrayDate:       "VT_ARRAY | VT_DATE",
		vtArrayBStr:       "VT_ARRAY | VT_BSTR",
		vtArrayError:      "VT_ARRAY | VT_ERROR",
		vtArrayBool:       "VT_ARRAY | VT_BOOL",
		vtArrayVariant:    "VT_ARRAY | VT_VARIANT",
		vtArrayDecimal:    "VT_ARRAY | VT_DECIMAL",
		vtArrayI1:         "VT_ARRAY | VT_I1",
		vtArrayUI1:        "VT_ARRAY | VT_UI1",
		vtArrayUI2:        "VT_ARRAY | VT_UI2",
		vtArrayUI4:        "VT_ARRAY | VT_UI4",
		vtArrayInt:        "VT_ARRAY | VT_INT",
		vtArrayUint:       "VT_ARRAY | VT_UINT",
	}
)

func parseExtraPropertyStore(size uint32, data []byte) (*PropertyStore, error) {
	if size < 0x0000000C {
		return nil, errors.New("invalid extra property store block size")
	}
	props := make(map[string][]Property)
	store := data[8:]
	offset := 0
	for {
		propertyData := store[offset:]
		if len(propertyData) < 4 {
			break
		}
		propertySize := binary.LittleEndian.Uint32(propertyData[0:4])
		if propertySize == 0 {
			break
		}
		if len(propertyData) < 24 || len(propertyData) < int(propertySize) {
			return nil, errors.New("invalid property size")
		}
		version := binary.LittleEndian.Uint32(propertyData[4:8])
		if version != 0x53505331 {
			return nil, errors.New("invalid property version")
		}
		format := encodeUUID(propertyData[8:24])
		name, properties, err := parseProperties(format, propertyData[24:propertySize])
		if err != nil {
			return nil, err
		}
		if properties != nil {
			props[name] = properties
		}
		offset += int(propertySize)
	}

	return &PropertyStore{
		Properties: props,
	}, nil
}

func parseProperties(identifier string, data []byte) (string, []Property, error) {
	propertySize := binary.LittleEndian.Uint32(data[0:4])
	if propertySize == 0 {
		return "", nil, nil
	}
	id := binary.LittleEndian.Uint32(data[4:8])
	name := identifier + "\\" + strconv.Itoa(int(id))
	knownFormat, known := knownProperties[identifier]
	if known {
		idName, knownName := knownFormat[id]
		if knownName {
			name = idName
		}
	}

	_, value, err := parseTypedValue(data[9:propertySize])
	if err != nil {
		return name, nil, err
	}
	return name, value, nil
}

func parseTypedValue(data []byte) (uint32, []Property, error) {
	if len(data) < 4 {
		return 0, nil, errors.New("invalid properties")
	}
	valueType := binary.LittleEndian.Uint32(data[0:4])
	switch valueType {
	case vtEmpty:
		fallthrough
	case vtNull:
		return valueType, []Property{
			Property{
				Type: propertyTypes[valueType],
			},
		}, nil
	case vtI2:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: int16(binary.LittleEndian.Uint16(data[4:8])),
			},
		}, nil
	case vtI4:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: int32(binary.LittleEndian.Uint32(data[4:8])),
			},
		}, nil
	case vtR4:
		bits := binary.LittleEndian.Uint32(data[4:8])
		float := math.Float32frombits(bits)
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: float,
			},
		}, nil
	case vtR8:
		bits := binary.LittleEndian.Uint64(data[4:12])
		float := math.Float64frombits(bits)
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: float,
			},
		}, nil
	case vtCY:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: binary.LittleEndian.Uint64(data[4:12]),
			},
		}, nil
	case vtDate:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: normalizeTime(binary.LittleEndian.Uint64(data[4:12])),
			},
		}, nil
	case vtBStr:
		codePageSize := binary.LittleEndian.Uint32(data[4:8])
		codePage := common.ReadString(data[8:8+codePageSize], 0)
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: codePage,
			},
		}, nil
	case vtError:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: binary.LittleEndian.Uint32(data[4:8]),
			},
		}, nil
	case vtBool:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: binary.LittleEndian.Uint16(data[4:6]) == 0xFFFF,
			},
		}, nil
	// case vtDecimal:
	// case vtI1:
	// case vtUI1:
	// case vtUI2:
	// case vtUI4:
	// case vtI8:
	// case vtUI8:
	// case vtInt:
	// case vtUInt:
	// case vtLPStr:
	case vtLPWStr:
		length := binary.LittleEndian.Uint32(data[4:8]) * 2
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: common.ReadUnicode(data[8:8+length], 0),
			},
		}, nil
	// case vtFiletime:
	// case vtBlob:
	// case vtStream:
	// case vtStorage:
	// case vtStreamedObject:
	// case vtStoredObject:
	// case vtBlobObject:
	// case vtCF:
	// case vtCLSID:
	// case vtVersionedStream:
	// case vtVectorI2:
	// case vtVectorI4:
	// case vtVectorR4:
	// case vtVectorR8:
	// case vtVectorCY:
	// case vtVectorDate:
	// case vtVectorBStr:
	// case vtVectorError:
	// case vtVectorBool:
	// case vtVectorVariant:
	// case vtVectorI1:
	// case vtVectorUI1:
	// case vtVectorUI2:
	// case vtVectorUI4:
	// case vtVectorI8:
	// case vtVectorUI8:
	// case vtVectorLPStr:
	// case vtVectorLPWStr:
	// case vtVectorFiletime:
	// case vtVectorCF:
	// case vtVectorCLSID:
	// case vtArrayI2:
	// case vtArrayI4:
	// case vtArrayR4:
	// case vtArrayR8:
	// case vtArrayCY:
	// case vtArrayDate:
	// case vtArrayBStr:
	// case vtArrayError:
	// case vtArrayBool:
	// case vtArrayVariant:
	// case vtArrayDecimal:
	// case vtArrayI1:
	// case vtArrayUI1:
	// case vtArrayUI2:
	// case vtArrayUI4:
	// case vtArrayInt:
	// case vtArrayUint:
	default:
		return valueType, []Property{
			Property{
				Type:  propertyTypes[valueType],
				Value: data[4:],
			},
		}, nil
	}
}
