FROM gcr.io/distroless/static-debian12:nonroot
COPY server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
