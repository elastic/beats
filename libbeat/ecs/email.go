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

package ecs

import "time"

// These fields are new top-level field set to facilitate email use cases `email.*`.
// The `email.*` field set adds fields for the sender, recipient, message header fields,
// and other attributes of an email message typically seen in logs produced by
// mail transfer agent (MTA) and email gateway applications.
type Email struct {

	// The date and time the email message was composed. Many email
	// clients will fill this in automatically when the message is
	// sent by a user.
	OriginationTimestamp time.Time `ecs:"origination_timestamp"`

	// The date and time the email message was received by the service
	// or client.
	DeliveryTimestamp time.Time `ecs:"delivery_timestamp"`

	// Stores the from email address from the RFC5322 From:
	// header field.
	FromAddress []string `ecs:"from.address"`

	// When the from field contains more than one address or the
	// sender and from are distinct then this field is populated.
	SenderAddress string `ecs:"sender.address"`

	// The email address of message recipient.
	ToAddress []string `ecs:"to.address"`

	// The email address of a carbon copy (CC) recipient.
	CcAddress []string `ecs:"cc.address"`

	// The email address of the blind carbon copy (CC) recipient(s).
	BccAddress []string `ecs:"bcc.address"`

	// The address that replies should be delivered to from the
	// RFC 5322 Reply-To: header field.
	ReplyToAddress []string `ecs:"reply_to.address"`

	// A brief summary of the topic of the message
	Subject string `ecs:"subject"`

	// Information about how the message is to be displayed.
	// Typically a MIME type
	ContentType string `ecs:"content_type"`

	// Identifier from the RFC5322 Message-ID: header field
	// that refers to a particular version of a particular message.
	MessageID string `ecs:"message_id"`

	// Unique identifier given to the email by the source
	// (MTA, gateway, etc.) that created the event and is not
	// persistent across hops (for example, the
	// X-MS-Exchange-Organization-Network-Message-Id id).
	LocalID string `ecs:"local_id"`

	// Direction of the message based on the sending and
	// receiving domains
	Direction string `ecs:"direction"`

	// What application was used to draft and send the original email.
	XMailer string `ecs:"x_mailer"`

	// Nested object of attachments on the email.
	Attachments []Attachments `ecs:"attachments"`
}

type Attachments struct {

	// MIME type of the attachment file.
	MimeType string `ecs:"file.mime_type"`

	// Name of the attachment file including the extension.
	FileName string `ecs:"file.name"`

	// Attachment file extension, excluding the leading dot.
	FileExtension string `ecs:"file.extension"`

	// Attachment file size in bytes.
	FileSize int `ecs:"file.size"`

	// Attachment file hash
	Hash Hash `ecs:"file.hash"`
}
