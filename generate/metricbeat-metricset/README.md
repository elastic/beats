This generator makes it possible to generate your own beat in seconds and build up on it.


# Create your own beat

To create your own beat based on this template, run inside your GOPATH where you want to create the beat:

```
cookiecutter https://github.com/ruflin/metricbeat-module-generator.git
```

This requires [python](https://www.python.org/downloads/) and [cookiecutter](https://github.com/audreyr/cookiecutter) to be installed. More details on how to install cookiecutter can be found [here](http://cookiecutter.readthedocs.io/en/latest/installation.html).


# Goals

This beat generator has several goals:

* Create a running metricbeat module in very few steps
* Have an environment for unit, integration and system testing ready to make testing a beat simple
* Ensure easy maintainable and standardised structure of a metricbeat module
* Provide all files needed to build up and grow a community around a metricbeat module
* Allow release management of a beat
* Make a beat easy to update to the most recent version of libbeat
