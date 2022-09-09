all : schemaV0.svg frameV0.svg schemaV1.svg frameV1.svg schemaV2.svg frameV2.svg

.PHONY : clean

%.svg : %.pic
	pikchr --svg-only $< > $@

clean :
	-rm *.svg
