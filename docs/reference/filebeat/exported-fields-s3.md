---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-s3.html
---

% This file is generated! See scripts/generate_fields_docs.py

# s3 fields [exported-fields-s3]

S3 fields from s3 input.

**`bucket.name`**
:   Name of the S3 bucket that this log retrieved from.

type: keyword


**`bucket.arn`**
:   ARN of the S3 bucket that this log retrieved from.

type: keyword


**`object.key`**
:   Name of the S3 object that this log retrieved from.

type: keyword


**`metadata`**
:   AWS S3 object metadata values.

type: flattened


