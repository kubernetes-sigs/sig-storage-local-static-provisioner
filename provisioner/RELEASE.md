# Release Process

local-volume-provisioner is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release
3. An OWNER runs `make test` to make sure tests pass
2. An OWNER runs `make push` with latest tag
4. An OWNER runs e2es with latest image to make sure tests pass
5. An OWNER runs `git tag -a $VERSION` and inserts the changelog and pushes the tag with `git push $VERSION`
6. An OWNER runs `make push` to build and push the image
7. The release issue is closed

