---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-openai.html
  # That link will 404 until 8.18 is current
  # (see https://www.elastic.co/guide/en/beats/metricbeat/8.18/exported-fields-openai.html)
---

# openai fields [exported-fields-openai]

openai module


## openai [_openai]


## usage [_usage_14]

OpenAI API usage metrics and statistics

**`openai.usage.organization_id`**
:   Organization identifier

type: keyword


**`openai.usage.organization_name`**
:   Organization name

type: keyword


**`openai.usage.api_key_id`**
:   API key identifier

type: keyword


**`openai.usage.api_key_name`**
:   API key name

type: keyword


**`openai.usage.api_key_redacted`**
:   Redacted API key

type: keyword


**`openai.usage.api_key_type`**
:   Type of API key

type: keyword


**`openai.usage.project_id`**
:   Project identifier

type: keyword


**`openai.usage.project_name`**
:   Project name

type: keyword



## data [_data_2]

General usage data metrics

**`openai.usage.data.requests_total`**
:   Number of requests made

type: long


**`openai.usage.data.operation`**
:   Operation type

type: keyword


**`openai.usage.data.snapshot_id`**
:   Snapshot identifier

type: keyword


**`openai.usage.data.context_tokens_total`**
:   Total number of context tokens used

type: long


**`openai.usage.data.generated_tokens_total`**
:   Total number of generated tokens

type: long


**`openai.usage.data.cached_context_tokens_total`**
:   Total number of cached context tokens

type: long


**`openai.usage.data.email`**
:   User email

type: keyword


**`openai.usage.data.request_type`**
:   Type of request

type: keyword



## dalle [_dalle]

DALL-E API usage metrics

**`openai.usage.dalle.num_images`**
:   Number of images generated

type: long


**`openai.usage.dalle.requests_total`**
:   Number of requests

type: long


**`openai.usage.dalle.image_size`**
:   Size of generated images

type: keyword


**`openai.usage.dalle.operation`**
:   Operation type

type: keyword


**`openai.usage.dalle.user_id`**
:   User identifier

type: keyword


**`openai.usage.dalle.model_id`**
:   Model identifier

type: keyword



## whisper [_whisper]

Whisper API usage metrics

**`openai.usage.whisper.model_id`**
:   Model identifier

type: keyword


**`openai.usage.whisper.num_seconds`**
:   Number of seconds processed

type: long


**`openai.usage.whisper.requests_total`**
:   Number of requests

type: long


**`openai.usage.whisper.user_id`**
:   User identifier

type: keyword



## tts [_tts]

Text-to-Speech API usage metrics

**`openai.usage.tts.model_id`**
:   Model identifier

type: keyword


**`openai.usage.tts.num_characters`**
:   Number of characters processed

type: long


**`openai.usage.tts.requests_total`**
:   Number of requests

type: long


**`openai.usage.tts.user_id`**
:   User identifier

type: keyword



## ft_data [_ft_data]

Fine-tuning data metrics

**`openai.usage.ft_data.original`**
:   Raw fine-tuning data

type: object



## assistant_code_interpreter [_assistant_code_interpreter]

Assistant Code Interpreter usage metrics

**`openai.usage.assistant_code_interpreter.original`**
:   Raw assistant code interpreter data

type: object



## retrieval_storage [_retrieval_storage]

Retrieval storage usage metrics

**`openai.usage.retrieval_storage.original`**
:   Raw retrieval storage data

type: object


