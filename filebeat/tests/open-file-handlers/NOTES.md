TODO:
* Check if truncation checks work es expected
* Test local folder on host, mounted by all


Questions:
* Should we use Data volumes or Data volume containers? https://docs.docker.com/engine/tutorials/dockervolumes/

Problems
* On volume per container: How do we add voumes dynamically to a container?
* Inodes are always reused on volumes -> typical on docker volumes as nothing else happens
