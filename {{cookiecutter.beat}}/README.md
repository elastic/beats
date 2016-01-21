# {{cookiecutter.beat|capitalize}}

Welcome to {{cookiecutter.beat}}.

To get running with your beat, run the following commands:

```
glide init
glide update
make update
make
```

To run your beat with debugging output enabled, run:

```
./{{cookiecutter.beat}} -c etc/{{cookiecutter.beat}}.yml -e -d "*"
```


To start it as a git repository, run

```
git init
```


# Create your own beat

To create your own beat based on this template, run inside your GOPATH where you want to create the beat:

```
cookiecutter github.com/elastic/beats
```

This requires python and cookiecutter to be installed (`pip install cookiecutter`).
