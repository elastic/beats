# Testing on OS X

To test the kubernetes module on Mac OS X you can use the following setup. [Minikube](https://github.com/kubernetes/minikube) is used for the testing and it is assumed that you have [brew](https://brew.sh/) installed.

First install minikube:

```
brew install Caskroom/cask/minikube
```

Start minikube exposing the metrics endpoint externally:

``` 
minikube start --extra-config apiserver.InsecureBindAddress=0.0.0.0
```

Now setup your metricbeat config to connect to the minikube kubernetes:

```
- module: kubernetes
  metricsets: ["node","container","volume","pod","system"]
  enabled: true
  period: 10s
  hosts: ["192.168.99.100:10255"]
```

Replace the IP address with the IP of your virtual box inside which minikube is running.
