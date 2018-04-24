XXX: this should be temporary until we can use the Elastic Registry.

To update the `tudorg/xgo-base` image on Docker HUB, do the following:

* `docker login` with your docker hub credentials. Feel free to publish the image
  to your own account if Tudor is not available.

* Build the image: `docker build --rm=true -t tudorg/xgo-base base/`

* List the images: `docker images`

* Tag it: `docker tag <image_id> tudorg/xgo-base:$(date '+v%Y%m%d')`

* Push: `docker push tudorg/xgo-base`

* Update `build.sh` to use the new image/tag
