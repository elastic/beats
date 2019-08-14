// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

/*
The Kibana CM Api returns a configuration format which cannot be ingested directly by our
configuration parser, it need to be transformed from the generic format into an adapted format
which is dependant on the type of configuration.


Translations:

Type: output

{
	  "success": true,
    "list": [

        {
            "config": {
              "_sub_type": "elasticsearch"
              "_id": "12312341231231"
              "hosts": [ "localhost" ],
              "password": "foobar"
              "username": "elastic"
            },
            "type": "output"
        }
    ]
}

YAML representation:

{
	"elasticsearch": {
		"hosts": [ "localhost" ],
		"password": "foobar"
		"username": "elastic"
	}
}


Type: *.modules

{
	  "success": true,
    "list": [
        {
            "config": {
              "_sub_type": "system"
              "_id": "12312341231231"
							"path" "foobar"
            },
            "type": "filebeat.module"
        }
    ]
}

YAML representation:

[
{
	"module": "system"
	"path": "foobar"
}
]

*/

package api
