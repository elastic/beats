package netflow

//go:generate go run fields_gen.go -output _meta/fields.yml --column-name=2 --column-type=3 --header _meta/fields.header.yml decoder/fields/ipfix-information-elements.csv
