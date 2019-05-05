FROM centos
COPY ./bin/wart /
COPY ./bin/wart.config /

CMD ["./wart", "--config=wart.config"]
