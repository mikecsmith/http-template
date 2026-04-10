FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
