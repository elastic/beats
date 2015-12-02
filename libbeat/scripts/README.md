The script directory contains various scripts which exist to automate everything around beats.
Below is a brief description of each file / folder.


| File / Folder        | Description                                                                                                                                            |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| docker               | Folder for all docker related files which are needed for the test environment                                                                          |
| crosscompile.bash    | Script to crosscompile a beat for all platforms. Use make crosscompile to use this script.                                                             |
| docker-entrypoint.sh | Entrypoint file used for the Dockerfile                                                                                                                |
| Makefile             | General Makefile which is copied over to all beats. This contains the basic methods which are shared across all beat                                   |
| update.sh            | This scripts brings a beat up-to-date based on the files in libbeat. This script should only be executed in a beat itself through running make update. |
