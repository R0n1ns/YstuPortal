FROM ubuntu:latest
LABEL authors="korob"

ENTRYPOINT ["top", "-b"]