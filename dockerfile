FROM centos
COPY ./bin/worker_unix /

ENTRYPOINT ["/worker_unix"]
