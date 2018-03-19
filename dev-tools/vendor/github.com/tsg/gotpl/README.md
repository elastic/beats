# gotpl - CLI tool for Golang templates

Command line tool that compiles Golang
[templates](http://golang.org/pkg/text/template/) with values from YAML files.

Inspired by Python/Jinja2's [j2cli](https://github.com/kolypto/j2cli).

## Install

    go get github.com/tsg/gotpl

## Usage

Say you have a `template` file like this:

    {{.first_name}} {{.last_name}} is {{.age}} years old.

and a `user.yml` YAML file like this one:

    first_name: Max
    last_name: Mustermann
    age: 30

You can compile the template like this:

    gotpl template < user.yml
