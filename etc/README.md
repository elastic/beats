The etc directory contains all config files which are needed to setup and run the beat.


| File                   	|                                                                                                                              	|
|------------------------	|------------------------------------------------------------------------------------------------------------------------------	|
| beat.yml               	| This is the base beat specific config file. All changes to the config file should be made here.                              	|
| fields.yml             	| This file contains all fields which are sent to elasticsearch. It is used to generate the filebeat.template.json file        	|
| filebeat.dev.yml       	| This file is ignored and not part of the repository. I can be used for local development                                     	|
| filebeat.template.json 	| This file is auto generated from fields.yml. Do not modify it.                                                               	|
| filebeat.yml           	| This is the full config which is shipped with the beat. It is automatically generated from beat.yml and libbeat config file. 	|
