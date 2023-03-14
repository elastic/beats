// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package decoder is a NetFlow and IPFIX Collector. It is fed NetFlow packets
// from the Exporter and outputs the records in an easy-to-use format.
//
// For legacy NetFlow versions (V1, V5, V6, V7, V8), it maps the static fields
// found in those protocols to existing NetFlow/IPFIX fields, which allows to
// work with any supported NetFlow version without worrying about the
// specific version that the Exporter is using.
//
// For more complex protocols (V9, IPFIX) it performs session/template management
// and expiration internally so the caller doesn't need to take care of
// maintaining sessions nor templates.
//
// # Status
//
// IPFIX
//
//   - Working implementation as of rfc7011.
//   - Options records supported.
//   - Variable-length fields supported.
//   - Missing: Support for RFC6313 data-types (basicList, subTemplateList, subTemplateMultiList).
//
// NetFlow 9
//
//   - Working implementation as of rfc3954.
//   - Support Options templates.
//
// NetFlow 8
//
//   - Supports the following aggregation types as defined in
//
// https://www.cisco.com/c/en/us/td/docs/net_mgmt/netflow_collection_engine/3-6/user/guide/format.html#wp1006730 :
//
//		RouterAS
//		RouterProtoPort
//		RouterSrcPrefix
//		RouterDstPrefix
//		RouterPrefix
//		DestOnly
//		SrcDst
//		FullFlow
//		TosAS
//		TosProtoPort
//		TosSrcPrefix
//		TosDstPrefix
//		TosPrefix
//		PrePortProtocol
//
//	 - Untested: Only validated by comparing to Wireshark decoder.
//
// NetFlow 6 & 7
//
//   - Untested: Only validated by comparing to Wireshark decoder.
//
// NetFlow 1 & 5
//
//   - Tested using softflowd
package decoder
