FROM mesosphere/mesos-master:1.6.2
RUN apt-get update
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:5050/metrics/snapshot | grep master
EXPOSE 5050
WORKDIR /sbin/ 
ENTRYPOINT ["mesos-master"]
CMD ["--ip=0.0.0.0", "--registry=in_memory", "--work_dir=/var/lib/mesos"]
